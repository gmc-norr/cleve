package gin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
)

var platformNovaSeq = cleve.Platform{
	Name:        "NovaSeq",
	ReadyMarker: "CopyComplete.txt",
}

var platformNextSeq = cleve.Platform{
	Name:        "NextSeq",
	ReadyMarker: "CopyComplete.txt",
}

func TestPlatformsHandler(t *testing.T) {
	gin.SetMode("test")
	pg := mock.PlatformGetter{}

	table := map[string]struct {
		Platforms cleve.Platforms
		Error     error
		Code      int
	}{
		"case 1": {
			cleve.Platforms{Platforms: []cleve.Platform{platformNovaSeq, platformNextSeq}},
			nil,
			200,
		},
		"case 2": {
			cleve.Platforms{},
			nil,
			200,
		},
	}

	for k, v := range table {
		pg.PlatformsFn = func() (cleve.Platforms, error) {
			return v.Platforms, v.Error
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		PlatformsHandler(&pg)(c)

		if !pg.PlatformsInvoked {
			t.Fatalf("Platforms not invoked for %s", k)
		}

		if w.Code != v.Code {
			t.Fatalf("Expected HTTP %d, got %d", v.Code, w.Code)
		}
	}
}

func TestGetPlatformHandler(t *testing.T) {
	gin.SetMode("test")
	pg := mock.PlatformGetter{}

	table := map[string]struct {
		Platform cleve.Platform
		Code     int
		Error    error
	}{
		"NovaSeq": {
			platformNovaSeq,
			200,
			nil,
		},
		"novaseq": {
			cleve.Platform{},
			404,
			mongo.ErrNoDocuments,
		},
	}

	for k, v := range table {
		pg.PlatformFn = func(name string) (cleve.Platform, error) {
			return v.Platform, v.Error
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		GetPlatformHandler(&pg)(c)

		if !pg.PlatformInvoked {
			t.Fatalf("Platform not invoked for %s", k)
		}

		if w.Code != v.Code {
			t.Fatalf("Expected HTTP %d, got %d for %s", v.Code, w.Code, k)
		}

		if w.Code == 200 {
			b, _ := io.ReadAll(w.Body)
			var p cleve.Platform
			if err := json.Unmarshal(b, &p); err != nil {
				t.Fatal(err)
			}
			if p.Name != v.Platform.Name {
				fmt.Printf("%#v != %#v", p, v.Platform)
				t.Fatalf("Incorrect response body for %s", k)
			}
			if len(p.Aliases) != len(v.Platform.Aliases) {
				fmt.Printf("%#v != %#v", p, v.Platform)
				t.Fatalf("Incorrect response body for %s", k)
			}
		}
	}
}
