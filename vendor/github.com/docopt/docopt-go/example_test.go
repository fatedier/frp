package docopt

import (
	"fmt"
	"sort"
)

func ExampleParse() {
	usage := `Usage:
  config_example tcp [<host>] [--force] [--timeout=<seconds>]
  config_example serial <port> [--baud=<rate>] [--timeout=<seconds>]
  config_example -h | --help | --version`
	// parse the command line `comfig_example tcp 127.0.0.1 --force`
	argv := []string{"tcp", "127.0.0.1", "--force"}
	arguments, _ := Parse(usage, argv, true, "0.1.1rc", false)
	// sort the keys of the arguments map
	var keys []string
	for k := range arguments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// print the argument keys and values
	for _, k := range keys {
		fmt.Printf("%9s %v\n", k, arguments[k])
	}
	// output:
	//    --baud <nil>
	//   --force true
	//    --help false
	// --timeout <nil>
	// --version false
	//        -h false
	//    <host> 127.0.0.1
	//    <port> <nil>
	//    serial false
	//       tcp true
}
