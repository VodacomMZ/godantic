# Getting Started

## Installation

```sh
go get github.com/VodacomMZ/godantic
```

## Basic Usage

```go
import "github.com/VodacomMZ/godantic"

var v godantic.Validate
err := v.BindJSON(jsonData, &myStruct)
```

This will bind and validate the JSON into your struct.