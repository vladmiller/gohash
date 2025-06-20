## Unique Hash of "almost" any Go value


Often times it may be desirable to retrieve a unique hash of a Golang variable, be it a simple scalar value,
or complex struct.

For example, it is often desirable to compute ETag for response objects.

GoHasher provides a simple interface that allows to retrieve a unique hash of almost any value in Golang, except
interfaces, channels and functions.

## Requirements

* Consistent & strict hashing of any Golang variable including type information, s.t. hash of `0` is not equal to hash of `false`. 
* Supports deeply nested maps and stuctures.
* Support for all common types, except interfaces, functions and channels.
* Handles Stringer interface
* Supports cyclic-references

## Installation

```bash
go get github.com/vladmiller/gohash
```

## Usage

```go
package main

func main() {

    hash, err := gohash.From("test", sha256.New())
    // ...

    hash, err := gohash.From(struct{}{}, sha256.New())
    // ...

    hash, err := gohash.From(nil, sha256.New())
    // ...

    hash, err := gohash.From(true, sha256.New())
    // ...

    hash, err := gohash.From(8.4434, sha256.New())
    // ...
}

```


## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=vladmiller/gohasher&type=Date)](https://www.star-history.com/#vladmiller/gohasher&Date)