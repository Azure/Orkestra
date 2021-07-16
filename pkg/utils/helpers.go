package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

func ConvertToDNS1123(in string) string {
	name := strings.ToLower(in)
	name = dns1123NotAllowedCharsRegexp.ReplaceAllString(name, "-")
	name = dns1123NotAllowedStartCharsRegexp.ReplaceAllString(name, "")

	if name == "" {
		name = GetHash(in)
	}
	return TruncateString(name, DNS1123NameMaximumLength)
}

func ConvertSliceToDNS1123(in []string) []string {
	out := []string{}
	for _, s := range in {
		out = append(out, ConvertToDNS1123(s))
	}
	return out
}

func GetHash(in string) string {
	h := sha256.New()
	h.Write([]byte(in))
	return hex.EncodeToString(h.Sum(nil))
}

func TruncateString(in string, num int) string {
	out := in
	if len(in) > num {
		out = in[0:num]
	}
	return out
}

func RemoveStringFromSlice(r string, s []string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func ToAnyString(in string) v1alpha13.AnyString {
	return v1alpha13.AnyString(in)
}

func ToAnyStringPtr(in string) *v1alpha13.AnyString {
	anystr := ToAnyString(in)
	return &anystr
}

func HrToYaml(hr fluxhelmv2beta1.HelmRelease) string {
	b, err := yaml.Marshal(hr)
	if err != nil {
		return ""
	}

	return string(b)
}

func HrToB64AnyStringPtr(hr *fluxhelmv2beta1.HelmRelease) *v1alpha13.AnyString {
	yaml := HrToYaml(*hr)
	base64 := base64.StdEncoding.EncodeToString([]byte(yaml))
	return ToAnyStringPtr(base64)
}

func TemplateContainsYaml(ch *chart.Chart) (bool, error) {
	if ch == nil {
		return false, fmt.Errorf("chart cannot be nil")
	}

	for _, f := range ch.Templates {
		if IsFileYaml(f.Name) {
			return true, nil
		}
	}
	return false, nil
}

func IsFileYaml(f string) bool {
	f = strings.ToLower(f)
	if strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml") {
		return true
	}
	return false
}

func AddAppChartNameToFile(name, a string) string {
	prefix := "templates/"
	name = strings.TrimPrefix(name, prefix)
	name = a + "_" + name
	return prefix + name
}
