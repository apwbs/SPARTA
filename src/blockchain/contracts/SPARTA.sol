// SPDX-License-Identifier: MIT
pragma solidity >= 0.5.0 < 0.9.0;

contract SPARTA {

  struct IPNSKey {
    address sender;
    bytes32 hashPart1;
    bytes32 hashPart2;
  }
  mapping (bytes32 => IPNSKey) allIPNSKeys;

  function setIPNSKey(bytes32 _keyName, bytes32 _hash1, bytes32 _hash2) public {
    allIPNSKeys[_keyName].sender = msg.sender;
    allIPNSKeys[_keyName].hashPart1 = _hash1;
    allIPNSKeys[_keyName].hashPart2 = _hash2;
  }

  function getIPNSKey(bytes32 _keyName) public view returns (address, bytes memory) {
    address sender = allIPNSKeys[_keyName].sender;
    bytes32 p1 = allIPNSKeys[_keyName].hashPart1;
    bytes32 p2 = allIPNSKeys[_keyName].hashPart2;
    bytes memory joined = new bytes(64);
    assembly {
      mstore(add(joined, 32), p1)
      mstore(add(joined, 64), p2)
    }
    return (sender, joined);
  }
}