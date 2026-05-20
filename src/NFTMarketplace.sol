// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "openzeppelin-contracts/contracts/token/ERC721/IERC721.sol";
import "openzeppelin-contracts/contracts/token/ERC1155/IERC1155.sol";
import "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

contract NFTMarketplace {
    enum TokenType { ERC721, ERC1155 }

    struct Listing {
        address seller;
        TokenType tokenType;
        uint256 amount;  // always 1 for ERC721
        uint256 price;   // unit price in tokenG
        bool active;
    }

    // nftContract => tokenId => Listing
    mapping(address => mapping(uint256 => Listing)) public listings;

    IERC20 public gameToken;

    event ItemListed(address indexed nftContract, uint256 indexed tokenId, address seller, uint256 amount, uint256 price);
    event ItemBought(address indexed nftContract, uint256 indexed tokenId, address buyer, uint256 amount, uint256 totalPrice);

    constructor(address _gameToken) {
        gameToken = IERC20(_gameToken);
    }

    // ─── ERC721 ────────────────────────────────────

    function listERC721(address nftContract, uint256 tokenId, uint256 price) external {
        require(IERC721(nftContract).ownerOf(tokenId) == msg.sender, "Not owner");
        IERC721(nftContract).approve(address(this), tokenId);

        listings[nftContract][tokenId] = Listing({
            seller: msg.sender,
            tokenType: TokenType.ERC721,
            amount: 1,
            price: price,
            active: true
        });
        emit ItemListed(nftContract, tokenId, msg.sender, 1, price);
    }

    function buyERC721(address nftContract, uint256 tokenId) external {
        Listing memory listing = listings[nftContract][tokenId];
        require(listing.active, "Not for sale");
        require(listing.tokenType == TokenType.ERC721, "Wrong token type");

        uint256 total = listing.price;
        require(gameToken.balanceOf(msg.sender) >= total, "Insufficient tokenG");
        require(gameToken.transferFrom(msg.sender, listing.seller, total), "Token transfer failed");

        IERC721(nftContract).transferFrom(listing.seller, msg.sender, tokenId);
        listings[nftContract][tokenId].active = false;

        emit ItemBought(nftContract, tokenId, msg.sender, 1, total);
    }

    // ─── ERC1155 ───────────────────────────────────

    /// @notice List `amount` copies of an ERC1155 token at unit price.
    function listERC1155(address nftContract, uint256 tokenId, uint256 amount, uint256 price) external {
        require(amount > 0, "Amount must be > 0");
        require(IERC1155(nftContract).balanceOf(msg.sender, tokenId) >= amount, "Not enough tokens");
        require(IERC1155(nftContract).isApprovedForAll(msg.sender, address(this)), "Marketplace not approved");

        listings[nftContract][tokenId] = Listing({
            seller: msg.sender,
            tokenType: TokenType.ERC1155,
            amount: amount,
            price: price,
            active: true
        });
        emit ItemListed(nftContract, tokenId, msg.sender, amount, price);
    }

    /// @notice Buy `amount` copies from an ERC1155 listing.
    function buyERC1155(address nftContract, uint256 tokenId, uint256 amount) external {
        Listing storage listing = listings[nftContract][tokenId];
        require(listing.active, "Not for sale");
        require(listing.tokenType == TokenType.ERC1155, "Wrong token type");
        require(amount > 0 && amount <= listing.amount, "Invalid amount");

        uint256 total = amount * listing.price;
        require(gameToken.balanceOf(msg.sender) >= total, "Insufficient tokenG");
        require(gameToken.transferFrom(msg.sender, listing.seller, total), "Token transfer failed");

        IERC1155(nftContract).safeTransferFrom(listing.seller, msg.sender, tokenId, amount, "");

        if (amount == listing.amount) {
            listing.active = false;
        } else {
            listing.amount -= amount;
        }

        emit ItemBought(nftContract, tokenId, msg.sender, amount, total);
    }

    /// @notice Cancel an active listing (seller only).
    function cancelListing(address nftContract, uint256 tokenId) external {
        Listing storage listing = listings[nftContract][tokenId];
        require(listing.active, "Not active");
        require(listing.seller == msg.sender, "Not seller");
        listing.active = false;
    }
}
