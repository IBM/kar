//go:generate swagger generate spec

package main

import "github.ibm.com/solsa/kar.git/core/internal/runtime"

func main() {
	runtime.Main()
}
