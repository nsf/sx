package main

import (
	"encoding/json"
	"fmt"
	"github.com/nsf/sx"
	"io/ioutil"
	"log"
	"os"
)

func astToJson(vs []sx.Node) []interface{} {
	out := []interface{}{}
	for _, v := range vs {
		if v.List != nil {
			out = append(out, astToJson(v.List))
		} else {
			out = append(out, v.Value)
		}
	}
	return out
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s <sx file>\n", os.Args[0])
		os.Exit(1)
	}
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("error reading file: %s", err)
	}

	ast, err := sx.Parse(data)
	if err != nil {
		log.Fatalf("error parsing sx file: %s", err)
	}

	js, err := json.MarshalIndent(astToJson(ast), "", "    ")
	if err != nil {
		log.Fatalf("error marshaling sx ast to json: %s", err)
	}

	fmt.Println(string(js))
}
