package sx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"testing"
)

type SChan struct {
	A chan int
}

type SValidSimple struct {
	Name  string `sx:"name"`
	Email string `sx:"email"`
}

type SValidBig struct {
	Min    int64
	Max    int64
	Path   string `sx:"path"`
	Digits []int  `sx:"digits"`
}

type S1 struct {
	Field *string
}

func stringPtr(s string) *string {
	ps := new(string)
	*ps = s
	return ps
}

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (v *Vec3) UnmarshalSX(tree []Node) error {
	if len(tree) != 3 {
		return errors.New("expected a list of 3 floating point elements")
	}
	var fs [3]float64
	for i, n := range tree {
		if !n.IsScalar() {
			return errors.New("expected a list of 3 floating point elements")
		}
		f, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return errors.New("expected a list of 3 floating point elements")
		}
		fs[i] = f
	}
	v.X = fs[0]
	v.Y = fs[1]
	v.Z = fs[2]
	return nil
}

type S2 struct {
	Int int16
}

type S3 struct {
	Uint uint16
}

type S4 struct {
	Float float64
}

type S5 struct {
	Bool bool
}

type S6 struct {
	Values []string
}

type S7 struct {
	Ints [4]int
}

type S8 struct {
	Map map[string]bool
}

type S9 struct {
	S8
}

type S10 struct {
	X byte `sx:"-"`
}

type S11 struct {
	x byte
}

var unmarshalCases = []struct {
	input    string
	schema   interface{}
	expected interface{}
	valid    bool
}{
	{"(A 10)", &SChan{}, nil, false},
	{`(name nsf) (email no.smile.face@gmail.com)`, &SValidSimple{}, &SValidSimple{"nsf", "no.smile.face@gmail.com"}, true},
	{`((name nsf) (email no.smile.face@gmail.com))`, &SValidSimple{}, &SValidSimple{"nsf", "no.smile.face@gmail.com"}, true},
	{`(((name nsf) (email no.smile.face@gmail.com)))`, &SValidSimple{}, &SValidSimple{"nsf", "no.smile.face@gmail.com"}, false},
	{"(Min 5) (Max 10) (Path `C:\\Program Files\\AntiVirus`) (Digits 3 1 4 1 5)", &SValidBig{},
		&SValidBig{
			Min:    5,
			Max:    10,
			Path:   `C:\Program Files\AntiVirus`,
			Digits: []int{3, 1, 4, 1, 5},
		}, true},
	{`(Field "Hello, World")`, &S1{}, &S1{stringPtr("Hello, World")}, true},
	{`(Field ("Hello, World"))`, &S1{}, nil, false},
	{`1.5 2.5 3.5`, &Vec3{}, &Vec3{1.5, 2.5, 3.5}, true},
	{`(1.5 2.5 3.5)`, &Vec3{}, &Vec3{1.5, 2.5, 3.5}, false},
	{`(Int 5)`, &S2{}, &S2{5}, true},
	{`(Int (5))`, &S2{}, nil, false},
	{`(Int abc)`, &S2{}, nil, false},
	{`(Int 1498723891729387129)`, &S2{}, nil, false},
	{`(Uint 5)`, &S3{}, &S3{5}, true},
	{`(Uint -5)`, &S3{}, nil, false},
	{`(Uint (5))`, &S3{}, nil, false},
	{`(Uint abc)`, &S3{}, nil, false},
	{`(Uint 1498723891729387129)`, &S3{}, nil, false},
	{`(Float 3.14)`, &S4{}, &S4{3.14}, true},
	{`(Float (5))`, &S4{}, nil, false},
	{`(Float abc)`, &S4{}, nil, false},
	{`(Float abc def)`, &S4{}, nil, false},
	{`(Bool true)`, &S5{}, &S5{true}, true},
	{`(Bool false)`, &S5{}, &S5{false}, true},
	{`(Bool (true))`, &S5{}, nil, false},
	{`(Bool maybe)`, &S5{}, nil, false},
	{`(Values abc def ghi)`, &S6{}, &S6{[]string{"abc", "def", "ghi"}}, true},
	{`(Values ((abc def ghi)))`, &S6{}, nil, false},
	{`(Ints 1 2)`, &S7{}, &S7{[4]int{1, 2, 0, 0}}, true},
	{`(Ints 1 2 3 4 5 6)`, &S7{}, &S7{[4]int{1, 2, 3, 4}}, true},
	{`(Map (edit true) (view false) (create true))`, &S8{}, &S8{map[string]bool{"edit": true, "view": false, "create": true}}, true},
	{`(Map (edit true) (view) (create true))`, &S8{}, nil, false},
	{`(Map (edit true) () (create true))`, &S8{}, nil, false},
	{`(Map (edit) () (create true))`, &S8{}, nil, false},
	{`(Map () (view false) (create true))`, &S8{}, nil, false},
	{`(Map edit (view false) (create true))`, &S8{}, nil, false},
	{`(Map (() true) (view false) (create true))`, &S8{}, nil, false},
	{`(Map (edit ()) (view false) (create true))`, &S8{}, nil, false},
	{`((Map) (edit ()) (view false) (create true))`, &S8{}, nil, false},
	{`(() (edit ()) (view false) (create true))`, &S8{}, nil, false},
	{`a b c`, &S8{}, nil, false},
	{`(S8 hello)`, &S9{}, &S9{}, true},
	{`(X hello)`, &S10{}, &S10{}, true},
	{`(x hello)`, &S11{}, &S11{}, false},
	{`(`, S11{}, &S11{}, false},
}

func prettyPrintAsJson(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func TestUnmarshal(t *testing.T) {
	for i, c := range unmarshalCases {
		err := Unmarshal([]byte(c.input), c.schema)
		if err != nil && c.valid {
			t.Errorf("case %d, unexpected error: %s", i, err)
			continue
		}
		if err == nil && !c.valid {
			t.Errorf("case %d, expected an error", i)
			continue
		}
		if c.valid && !reflect.DeepEqual(c.schema, c.expected) {
			t.Errorf("case %d\ngot:\n%s\nexpected:\n%s", i, prettyPrintAsJson(c.schema), prettyPrintAsJson(c.expected))
		}
	}
}

type PortMapping struct {
	ContainerPort int    `sx:"containerPort"`
	HostPort      int    `sx:"hostPort"`
	ServicePort   int    `sx:"servicePort"`
	Protocol      string `sx:"protocol"`
}

type Parameter struct {
	Key   string `sx:"key"`
	Value string `sx:"value"`
}

type Docker struct {
	Image        string        `sx:"image"`
	Network      string        `sx:"network"`
	PortMappings []PortMapping `sx:"portMappings"`
	Privileged   bool          `sx:"privileged"`
	Parameters   []Parameter   `sx:"parameters"`
}

type Volume struct {
	ContainerPath string `sx:"containerPath"`
	HostPath      string `sx:"hostPath"`
	Mode          string `sx:"mode"`
}

type Container struct {
	Type    string   `sx:"type"`
	Docker  *Docker  `sx:"docker"`
	Volumes []Volume `sx:"volumes"`
}

type Command struct {
	Value string `sx:"value"`
}

type HealthCheck struct {
	Protocol               string   `sx:"protocol"`
	Path                   string   `sx:"path"`
	GracePeriodSeconds     int      `sx:"gracePeriodSeconds"`
	IntervalSeconds        int      `sx:"intervalSeconds"`
	PortIndex              int      `sx:"portIndex"`
	TimeoutSeconds         int      `sx:"timeoutSeconds"`
	MaxConsecutiveFailures int      `sx:"maxConsecutiveFailures"`
	Command                *Command `sx:"command"`
}

type UpgradeStrategy struct {
	MinimumHealthCapacity float64 `sx:"minimumHealthCapacity"`
	MaximumOverCapacity   float64 `sx:"maximumOverCapacity"`
}

type MarathonConfig struct {
	Id                      string            `sx:"id"`
	Cmd                     string            `sx:"cmd"`
	Args                    []string          `sx:"args"`
	CPUs                    float64           `sx:"cpus"`
	Mem                     float64           `sx:"mem"`
	Ports                   []int             `sx:"ports"`
	RequirePorts            bool              `sx:"requirePorts"`
	Instances               int               `sx:"instances"`
	Executor                string            `sx:"executor"`
	Container               *Container        `sx:"container"`
	Env                     map[string]string `sx:"env"`
	Constraints             [][]string        `sx:"constraints"`
	AcceptableResourceRoles []string          `sx:"acceptableResourceRoles"`
	Labels                  map[string]string `sx:"labels"`
	Uris                    []string          `sx:"uris"`
	Dependencies            []string          `sx:"dependencies"`
	HealthChecks            []HealthCheck     `sx:"healthChecks"`
	BackoffSeconds          int               `sx:"backoffSeconds"`
	BackoffFactor           float64           `sx:"backoffFactor"`
	MaxLaunchDelaySeconds   int               `sx:"maxLaunchDelaySeconds"`
	UpgradeStrategy         *UpgradeStrategy  `sx:"upgradeStrategy"`
}

func TestMarathonConfig(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/marathon.sx")
	if err != nil {
		t.Fatal(err)
	}
	var config MarathonConfig
	err = Unmarshal(data, &config)
	fmt.Println(prettyPrintAsJson(config))
}
