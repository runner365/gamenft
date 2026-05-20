// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";
import "openzeppelin-contracts/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "openzeppelin-contracts/contracts/access/Ownable.sol";

/**
 * @title GameToken
 * @dev Simple ERC20 token for in-game item trading. Owner can mint; holders can burn.
 */
contract GameToken is ERC20, ERC20Burnable, Ownable {
    uint256 public rate; // GMTK per 1 ETH (in token smallest units)

    event TokensPurchased(address indexed buyer, uint256 ethAmount, uint256 tokenAmount);
    event RateUpdated(uint256 oldRate, uint256 newRate);

    constructor(string memory name_, string memory symbol_, uint256 initialSupply_) ERC20(name_, symbol_) {
        _mint(_msgSender(), initialSupply_);
        rate = 1000 * 10**18; // 1 ETH = 1000 GMTK
    }

    /**
     * @dev Owner can mint new tokens to `to`.
     */
    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    /**
     * @dev Buy GMTK by sending ETH. Token amount = msg.value * rate / 1 ether.
     */
    function buyTokens() external payable {
        _buyTokens();
    }

    /**
     * @dev Fallback — sending ETH directly to the contract buys GMTK.
     */
    receive() external payable {
        _buyTokens();
    }

    /**
     * @dev Internal buy logic shared by buyTokens() and receive().
     */
    function _buyTokens() private {
        require(msg.value > 0, "Send ETH to buy tokens");
        uint256 tokenAmount = (msg.value * rate) / 1 ether;
        require(tokenAmount > 0, "ETH amount too low");
        _mint(msg.sender, tokenAmount);
        emit TokensPurchased(msg.sender, msg.value, tokenAmount);
    }

    /**
     * @dev Owner sets the exchange rate (GMTK per 1 ETH, in token smallest units).
     */
    function setRate(uint256 newRate) external onlyOwner {
        require(newRate > 0, "Rate must be > 0");
        emit RateUpdated(rate, newRate);
        rate = newRate;
    }

    /**
     * @dev Owner withdraws accumulated ETH from the contract.
     */
    function withdrawETH() external onlyOwner {
        (bool ok, ) = payable(owner()).call{value: address(this).balance}("");
        require(ok, "ETH withdrawal failed");
    }
}
