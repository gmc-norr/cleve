package gin

import (
	"bytes"
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

var platformNovaSeq = &cleve.Platform{
	Name:        "NovaSeq",
	ReadyMarker: "CopyComplete.txt",
}

var platformNovaSeqAdd = &cleve.Platform{
	Name: "NovaSeq",
}

var platformNextSeq = &cleve.Platform{
	Name:        "NextSeq",
	ReadyMarker: "CopyComplete.txt",
}

func TestPlatformsHandler(t *testing.T) {
	gin.SetMode("test")
	pg := mock.PlatformGetter{}

	table := map[string]struct {
		Platforms []*cleve.Platform
		Error     error
		Code      int
	}{
		"case 1": {
			[]*cleve.Platform{platformNovaSeq, platformNextSeq},
			nil,
			200,
		},
		"case 2": {
			[]*cleve.Platform{},
			nil,
			200,
		},
	}

	for k, v := range table {
		pg.PlatformsFn = func() ([]*cleve.Platform, error) {
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
		Platform *cleve.Platform
		Code     int
		Error    error
	}{
		"NovaSeq": {
			platformNovaSeq,
			200,
			nil,
		},
		"novaseq": {
			nil,
			404,
			mongo.ErrNoDocuments,
		},
	}

	for k, v := range table {
		pg.PlatformFn = func(name string) (*cleve.Platform, error) {
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
			if p != *v.Platform {
				fmt.Printf("%#v != %#v", p, v.Platform)
				t.Fatalf("Incorrect response body for %s", k)
			}
		}
	}
}

func TestAddPlatformHandler(t *testing.T) {
	gin.SetMode("test")
	table := map[string]struct {
		RequestPlatform *cleve.Platform
		Code            int
		InvokesCreate   bool
		Error           error
	}{
		"novaseq1": {
			RequestPlatform: platformNovaSeq,
			Code:            200,
			InvokesCreate:   true,
		},
		"novaseq_no_readymarker": {
			RequestPlatform: platformNovaSeqAdd,
			Code:            200,
			InvokesCreate:   true,
		},
		"duplicate_platform": {
			RequestPlatform: platformNovaSeq,
			Code:            409,
			InvokesCreate:   true,
			Error:           mongo.GenericDuplicateKeyError,
		},
	}

	for k, v := range table {
		t.Run(k, func(t *testing.T) {
			ps := mock.PlatformSetter{}
			ps.CreatePlatformFn = func(p *cleve.Platform) error {
				return v.Error
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			rBody, _ := json.Marshal(v.RequestPlatform)
			r := httptest.NewRequest("POST", "/api/platforms", bytes.NewReader(rBody))
			c.Request = r

			AddPlatformHandler(&ps)(c)

			if ps.CreatePlatformInvoked != v.InvokesCreate {
				if v.InvokesCreate {
					t.Fatalf("CreatePlatform should have been invoked, but was not")
				} else {
					t.Fatalf("CreatePlatform should not have been invoked, but was")
				}
			}

			if w.Code != v.Code {
				t.Logf("%s", w.Body.String())
				t.Fatalf("Expected HTTP %d, got %d for %s", v.Code, w.Code, k)
			}
		})
	}
}
