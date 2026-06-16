// Package greeter builds greeting strings. It has no main function, so it
// cannot run on its own — it is a library, imported by a command.
//
// The import path of this package is the module path + the directory:
//   cxm/m00/greeter  (module)  +  /greeter  (dir)  =  cxm/m00/greeter/greeter
package greeter

import "fmt"

// Greet returns a greeting for name. Exported (capitalized) so other packages
// can call it; an empty name greets the world.
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("hello, %s", name)
}
