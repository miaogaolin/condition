Minimalist rules engine for Golang.

## Usage Example

```go
package main

import (
    "fmt"

    "github.com/miaogaolin/condition"
)

func main() {
    data := map[string]interface{}{
		"col1": 1,
		"col2": "hello world",
		"col3": "male",
	}
	res, err := condition.Validate(data, `({col1}==1 and {col2} =~ "world") or {col3} in ["male"]`)
	if err != nil {
		panic(err)
	}

    if res {
        fmt.Println("success")
    }
}
```
# Other Symbols

```go
== 
> 
< 
>=
<=
!=
in  
not in 
=~  // Contains
!= // Not contained
```