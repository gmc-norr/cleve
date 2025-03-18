package gin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading platform information from the database.
type PlatformGetter interface {
	Platform(string) (cleve.Platform, error)
	Platforms() (cleve.Platforms, error)
}

// Interface for storing platforms in the database.
type PlatformSetter interface {
	CreatePlatform(*cleve.Platform) error
}

func PlatformsHandler(db PlatformGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		platforms, err := db.Platforms()
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err},
			)
			return
		}

		c.JSON(http.StatusOK, platforms)
	}
}

func GetPlatformHandler(db PlatformGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("platformName")

		platform, err := db.Platform(name)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(
					http.StatusNotFound,
					gin.H{"error": "no such platform", "name": name},
				)
			}
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.JSON(http.StatusOK, platform)
	}
}

func AddPlatformHandler(db PlatformSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var platform cleve.Platform

		if err := c.BindJSON(&platform); err != nil {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"error": err.Error()},
			)
			return
		}

		if platform.ReadyMarker == "" {
			platform.ReadyMarker = "CopyComplete.txt"
		}

		if err := db.CreatePlatform(&platform); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				c.AbortWithStatusJSON(
					http.StatusConflict,
					gin.H{"error": fmt.Sprintf("platform %s already exists", platform.Name)},
				)
				return
			}
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.Status(http.StatusOK)
	}
}
