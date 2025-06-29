# Codec Support Functions

## Purpose

Provides factory functions for the codec package to create node instances from decoded data. These functions maintain encapsulation while allowing the codec to reconstruct nodes without direct access to private fields.

## Design Decisions

- **Separate from Constructors**: Unlike NewLeafNode which validates and hashes values, these assume pre-validated data from storage
- **No Validation**: Assumes data from storage is already valid (was validated on write)
- **Minimal State**: Only sets essential fields, caches are computed on demand
- **Version Required**: Version must be provided as it's not part of the serialized data

## Usage Pattern

These functions are only intended for use by the codec package when deserializing nodes from storage. Regular code should use the standard constructors (NewLeafNode, NewInternalNode).