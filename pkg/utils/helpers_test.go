package utils

import (
	"reflect"
	"testing"
)

func TestConvertToDNS1123(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testing valid name",
			args: args{
				in: "good-small-name",
			},
			want: "good-small-name",
		},
		{
			name: "testing invalid name",
			args: args{
				in: "tOk3_??ofTHE-Runner",
			},
			want: "tok3---ofthe-runner",
		},
		{
			name: "testing all characters are invalid",
			args: args{
				in: "?.?&^%#$@_??",
			},
			want: GetHash("?.?&^%#$@_??")[0:63],
		},
		{
			name: "testing invalid start chars",
			args: args{
				in: "----??tOk3_??ofTHE-Runner",
			},
			want: "tok3---ofthe-runner",
		},
		{
			name: "testing very long name",
			args: args{
				in: "very-long-name------------------------------------------------end",
			},
			want: "very-long-name------------------------------------------------e",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToDNS1123(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertDNS1123() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	type args struct {
		in  string
		num int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testing empty",
			args: args{
				in:  "",
				num: 0,
			},
			want: "",
		},
		{
			name: "testing empty with > 0 requested length",
			args: args{
				in:  "",
				num: 5,
			},
			want: "",
		},
		{
			name: "testing input string length == requested length",
			args: args{
				in:  "hello, don't truncate me",
				num: 24,
			},
			want: "hello, don't truncate me",
		},
		{
			name: "testing input string length < requested length",
			args: args{
				in:  "hello again, don't truncate me",
				num: 63,
			},
			want: "hello again, don't truncate me",
		},
		{
			name: "testing input string length > requested length",
			args: args{
				in:  "truncate_this_string_so_that_its_length_is_less_than_sixty_three_characters",
				num: 63,
			},
			want: "truncate_this_string_so_that_its_length_is_less_than_sixty_thre",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateString(tt.args.in, tt.args.num); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TruncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFileYaml(t *testing.T) {
	type args struct {
		f string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "testing empty",
			args: args{
				f: "",
			},
			want: false,
		},
		{
			name: "testing .yaml file",
			args: args{
				f: "templates/filename.yaml",
			},
			want: true,
		},
		{
			name: "testing .yml file",
			args: args{
				f: "templates/bin/myfile.yml",
			},
			want: true,
		},
		{
			name: "testing .yaml file",
			args: args{
				f: "templates/bin/myfile.txt",
			},
			want: false,
		},
		{
			name: "testing filename extension that contains yaml but not yaml file",
			args: args{
				f: "templates/bin/myfile.myyaml",
			},
			want: false,
		},
		{
			name: "testing .txt file with name containing yaml substring.",
			args: args{
				f: "templates/bin/yamlFileGuide.txt",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFileYaml(tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IsFileYaml() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveStringFromSlice(t *testing.T) {
	type args struct {
		s string
		v []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "removing with single item",
			args: args{
				s: "apple",
				v: []string{"apple"},
			},
			want: []string{},
		},
		{
			name: "removing from middle of slice",
			args: args{
				s: "orange",
				v: []string{"apple", "orange", "banana"},
			},
			want: []string{"apple", "banana"},
		},
		{
			name: "not finding the item in the slice",
			args: args{
				s: "papaya",
				v: []string{"apple", "orange", "banana"},
			},
			want: []string{"apple", "orange", "banana"},
		},
		{
			name: "finding item at the end of slice",
			args: args{
				s: "banana",
				v: []string{"apple", "orange", "banana"},
			},
			want: []string{"apple", "orange"},
		},
		{
			name: "passing empty slice",
			args: args{
				s: "banana",
				v: []string{},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveStringFromSlice(tt.args.s, tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IsFileYaml() = %v, want %v", got, tt.want)
			}
		})
	}
}
