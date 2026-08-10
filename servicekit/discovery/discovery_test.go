package discovery

import (
	"reflect"
	"testing"
)

func TestWatchPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		watchAll bool
		want     []string
	}{
		{name: "microservice watches nothing", services: []string{}, want: []string{}},
		{name: "microservice watches configured services", services: []string{"user", "computing"}, want: []string{"/one_adventure/computing/", "/one_adventure/user/"}},
		{name: "gateway watches all", watchAll: true, want: []string{"/one_adventure/"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := watchPrefixes(test.services, test.watchAll); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("watchPrefixes() = %#v, want %#v", got, test.want)
			}
		})
	}
}
