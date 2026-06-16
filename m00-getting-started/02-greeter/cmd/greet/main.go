// Command greet prints a greeting. It demonstrates two things:
//   - the cmd/<name>/ convention for executables, and
//   - importing a library package from the same module.
//
// Run: go run ./cmd/greet -name Ada
package main

import (
	"flag"
	"fmt"

	"cxm/m00/greeter/greeter"
)

func main() {
	name := flag.String("name", "", "who to greet")
	flag.Parse()
	fmt.Println(greeter.Greet(*name))
}
