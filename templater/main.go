package main

import "flag"

var (
	tmpl string
	sub  string
)

func main() {
	flag.StringVar(&tmpl, "tmpl", "tmpl.rb", "filename of template")
	flag.StringVar(&sub, "sub", "puts hi",
		"what to substitute into the template")
	flag.Parse()
}
