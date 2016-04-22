package formam

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"testing"
	"time"
)

type Text string

func (s *Text) UnmarshalText(text []byte) error {
	var n Text
	n = "the string has changed by UnmarshalText method"
	*s = n
	return nil
}

type UUID [16]byte

func (u *UUID) UnmarshalText(text []byte) error {
	if len(text) != 32 {
		return fmt.Errorf("text must be exactly 16 bytes long, got %d bytes", len(text))
	}
	_, err := hex.Decode(u[:], text)
	if err != nil {
		return err
	}
	return nil
}

type Anonymous struct {
	Int            int
	AnonymousField string
}

type PtrStruct struct {
	String *string
}

type TestStruct struct {
	Anonymous
	Int  []string
	Nest struct {
		Children []struct {
			ID   string
			Name string
		}
	}
	String       string
	Slice        []int
	MapSlice     map[string][]string
	MapMap       map[string]map[string]string
	MapRecursive map[string]map[string]map[string]struct {
		Recursive bool
	}
	Bool            bool
	Ptr             *string
	Tag             string `formam:"tag"`
	Time            time.Time
	URL             url.URL
	PtrStruct       *PtrStruct
	UnmarshalText   Text
	MapCustomKey    map[UUID]string
	MapCustomKeyPtr map[*UUID]string
	InterfaceStruct interface{}
	Interface       interface{}
}

type InterfaceStruct struct {
	ID   int
	Name string
}

var structValues = url.Values{
	"Nest.Children[0].ID":                              []string{"monoculum_id"},
	"Nest.Children[0].Name":                            []string{"Monoculum"},
	"MapSlice.names[0]":                                []string{"shinji"},
	"MapSlice.names[2]":                                []string{"sasuka"},
	"MapSlice.names[4]":                                []string{"carla"},
	"MapSlice.countries[0]":                            []string{"japan"},
	"MapSlice.countries[1]":                            []string{"spain"},
	"MapSlice.countries[2]":                            []string{"germany"},
	"MapSlice.countries[3]":                            []string{"united states"},
	"MapMap.titles.es_es":                              []string{"El viaje de Chihiro"},
	"MapMap.titles.en_us":                              []string{"The spirit away"},
	"MapRecursive.map.struct.are.Recursive":            []string{"true"},
	"Slice[0]":                                         []string{"1"},
	"Slice[1]":                                         []string{"2"},
	"Int[0]":                                           []string{"10"}, // Int is located inside Anonymous struct
	"AnonymousField":                                   []string{"anonymous!"},
	"Bool":                                             []string{"true"},
	"tag":                                              []string{"tagged"},
	"Ptr":                                              []string{"this is a pointer to string"},
	"Time":                                             []string{"2006-10-08"},
	"URL":                                              []string{"https://www.golang.org"},
	"PtrStruct.String":                                 []string{"dashaus"},
	"UnmarshalText":                                    []string{"unmarshal text"},
	"MapCustomKey.11e5bf2d3e403a8c86740023dffe5350":    []string{"Princess Mononoke"},
	"MapCustomKeyPtr.11e5bf2d3e403a8c86740023dffe5350": []string{"*Princess Mononoke"},
	"InterfaceStruct.ID":                               []string{"1"},
	"InterfaceStruct.Name":                             []string{"Go"},
	"Interface":                                        []string{"only interface"},
}

func TestDecodeInStruct(t *testing.T) {
	var t1 TestStruct
	t1.InterfaceStruct = &InterfaceStruct{}
	err := Decode(structValues, &t1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("RESULT: ", t1)
}

type TestSlice []string

var sliceValues = url.Values{
	"[0]": []string{"spanish"},
	"[1]": []string{"english"},
}

func TestDecodeInSlice(t *testing.T) {
	var t2 TestSlice
	err := Decode(sliceValues, &t2)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("RESULT: ", t2)
}
