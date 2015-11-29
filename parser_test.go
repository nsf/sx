package sx

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func prettyPrint(ast []Node) string {
	data, err := json.MarshalIndent(ast, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func expect(strs ...string) []Node {
	var nodes []Node
	for _, str := range strs {
		nodes = append(nodes, Node{nil, str})
	}
	return nodes
}

func convertJsonToAst(v []interface{}) []Node {
	out := []Node{}
	for _, elem := range v {
		switch e := elem.(type) {
		case string:
			out = append(out, Node{nil, e})
		case []interface{}:
			out = append(out, Node{convertJsonToAst(e), ""})
		default:
			panic("invalid json type")
		}
	}
	return out
}

func convertAstToJson(v []Node) []interface{} {
	out := []interface{}{}
	for _, elem := range v {
		if elem.List != nil {
			out = append(out, convertAstToJson(elem.List))
		} else {
			out = append(out, elem.Value)
		}
	}
	return out
}

func expectJson(str string) []Node {
	var out []interface{}
	err := json.Unmarshal([]byte(str), &out)
	if err != nil {
		panic(err)
	}
	return convertJsonToAst(out)
}

var cases = []struct {
	valid    bool
	input    string
	expected []Node
}{
	// 0
	{true, "hello world", expect("hello", "world")},
	{true, ";hello", nil},
	{true, ";hello\n;world\n\n\n;xxx", nil},
	{true, ";hello\n;world\n\n\n;xxx\nabc\ndef", expect("abc", "def")},
	{true, "        \n\n\r\n\t", nil},

	// 5
	{true, "", nil},
	{false, "\"abc", nil},
	{false, "\"", nil},
	{false, "\"\n", nil},
	{false, "\"abc\n", nil},

	// 10
	{true, `"hello, world" abc`, expect("hello, world", "abc")},
	{true, `"\r\n"`, expect("\r\n")},
	{false, `"\N"`, nil},
	{false, `"\xFX"`, nil},
	{false, `"\xfX"`, nil},

	// 15
	{true, `"\xff"`, expect("\xff")},
	{true, `"\xaF"`, expect("\xaf")},
	{true, `"\xFb"`, expect("\xfb")},
	{true, `"\x42"`, expect("\x42")},
	{false, `"\x5`, nil},

	// 20
	{true, "`\\n`", expect(`\n`)},
	{false, "` \n`", nil},
	{false, "`", nil},
	{false, "`\n", nil},
	{true, "`hello, \\xFF`", expect(`hello, \xFF`)},

	// 25
	{false, "`\nxxx`", nil},
	{true, "`\n|xxx\n`", expect(`xxx`)},
	{true, "`\r\n|xxx\r\n\r\n\t\t\n  \t|yyy\r\n`", expect("xxx\nyyy")},
	{false, "`\n|xxx`", nil},
	{true, "(hello world)", expectJson(`[["hello", "world"]]`)},

	// 30
	{false, ")hello", nil},
	{false, "(hello) )", nil},
	{true, "(123 (\n\t456 789   ) foo)", expectJson(`[["123", ["456", "789"], "foo"]]`)},
	{true, "(123 (\n\t456 789 ; xxx\n zzz  ) foo)", expectJson(`[["123", ["456", "789", "zzz"], "foo"]]`)},
	{true, "123 (\n\t456 789 ; xxx\n zzz  ) foo", expectJson(`["123", ["456", "789", "zzz"], "foo"]`)},

	// 35
	{false, "12 (34 (56 (78 (9", nil},
	{true, "12(34(56`hello`\"world\"))", expectJson(`["12", ["34", ["56", "hello", "world"]]]`)},
	{true, "()", expectJson(`[[]]`)},
	{true, `hello(iam"John")world`, expectJson(`["hello", ["iam", "John"], "world"]`)},
}

func TestParser(t *testing.T) {
	for i, c := range cases {
		result, err := Parse([]byte(c.input))
		if err != nil && c.valid {
			t.Errorf("case %d, unexpected error: %s", i, err)
			continue
		}
		if err == nil && !c.valid {
			t.Errorf("case %d, expected an error", i)
			continue
		}
		if !reflect.DeepEqual(result, c.expected) {
			t.Errorf("case %d\ngot:\n%s\nexpected:\n%s", i, prettyPrint(result), prettyPrint(c.expected))
		}
	}

	testfiles, err := filepath.Glob("testdata/*.sx")
	if err != nil {
		t.Error(err)
		return
	}
	for _, sxname := range testfiles {
		jsname := sxname[:len(sxname)-2] + "json"
		sxdata, err := ioutil.ReadFile(sxname)
		if err != nil {
			t.Error(err)
			continue
		}
		jsondata, err := ioutil.ReadFile(jsname)
		if err != nil {
			t.Error(err)
			continue
		}

		var js []interface{}
		err = json.Unmarshal(jsondata, &js)
		if err != nil {
			t.Errorf("error parsing '%s' json file: %s", jsname, err)
		}

		sx, err := Parse(sxdata)
		if err != nil {
			t.Errorf("error parsing '%s' sx file: %s", sxname, err)
			continue
		}

		sxjs := convertAstToJson(sx)
		if !reflect.DeepEqual(js, sxjs) {
			t.Errorf("files '%s' and '%s' do not produce same trees", sxname, jsname)
		}
	}
}
