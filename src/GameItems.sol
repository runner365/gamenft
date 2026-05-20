// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "openzeppelin-contracts/contracts/token/ERC1155/ERC1155.sol";
import "openzeppelin-contracts/contracts/access/Ownable.sol";
import "openzeppelin-contracts/contracts/utils/Strings.sol";
import "openzeppelin-contracts/contracts/security/ReentrancyGuard.sol";

/**
 * @title GameItems
 * @dev ERC1155 multi-token for in-game items (knife/pistol/bomb).
 *      One tokenId per item type; players hold fungible amounts.
 */
contract GameItems is ERC1155, Ownable, ReentrancyGuard {
    uint256 public constant KNIFE  = 1;
    uint256 public constant PISTOL = 2;
    uint256 public constant BOMB   = 3;

    string private _baseURI;

    // tokenId => max supply (0 = unlimited)
    mapping(uint256 => uint256) public maxSupply;
    // tokenId => total circulated supply (minted - burned)
    mapping(uint256 => uint256) public totalSupply;

    constructor(string memory baseURI_) ERC1155("") Ownable() {
        _baseURI = baseURI_;
    }

    // ─── mint ──────────────────────────────────────

    /// @notice Owner mints `amount` of `id` to `to`.
    ///         Follows CEI: checks → effects → interactions.
    function mint(address to, uint256 id, uint256 amount) external onlyOwner nonReentrant {
        if (maxSupply[id] > 0) {
            require(totalSupply[id] + amount <= maxSupply[id], "Exceeds max supply");
        }
        // effects
        totalSupply[id] += amount;
        // interactions — _mint may call onERC1155Received on `to`
        _mint(to, id, amount, "");
    }

    /// @notice Owner batch-mints multiple ids to `to`.
    function mintBatch(address to, uint256[] calldata ids, uint256[] calldata amounts) external onlyOwner nonReentrant {
        require(ids.length == amounts.length, "Length mismatch");
        for (uint256 i = 0; i < ids.length; i++) {
            if (maxSupply[ids[i]] > 0) {
                require(totalSupply[ids[i]] + amounts[i] <= maxSupply[ids[i]], "Exceeds max supply");
            }
            totalSupply[ids[i]] += amounts[i];
        }
        _mintBatch(to, ids, amounts, "");
    }

    // ─── burn ──────────────────────────────────────

    /// @notice Player burns their own items (in-game use).
    ///         nonReentrant protects against reentrancy via ERC1155 hooks.
    function burn(address from, uint256 id, uint256 amount) external nonReentrant {
        require(from == msg.sender || isApprovedForAll(from, msg.sender), "Not approved");
        // effects
        totalSupply[id] -= amount;
        // interactions
        _burn(from, id, amount);
    }

    // ─── admin ────────────────────────────────────

    function setMaxSupply(uint256 id, uint256 supply) external onlyOwner {
        require(supply >= totalSupply[id], "Supply below current total");
        maxSupply[id] = supply;
    }

    function setBaseURI(string memory uri_) external onlyOwner {
        _baseURI = uri_;
    }

    // ─── metadata ─────────────────────────────────

    function uri(uint256 id) public view override returns (string memory) {
        return string(abi.encodePacked(_baseURI, Strings.toString(id), ".json"));
    }
}
