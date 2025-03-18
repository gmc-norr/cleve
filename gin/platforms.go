package gin

import (
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

		c.JSON(http.StatusOK, platforms.Platforms)
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
				return
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
