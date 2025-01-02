// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/boyter/scc/v3/processor"
	"github.com/gin-gonic/gin"
)

func scanHandler(c *gin.Context) {
	rawPath := c.Request.URL.Path

	validIdentifier := regexp.MustCompile(`^/[^/]+/[^/]+`)
	path := validIdentifier.FindString(rawPath)
	if path == "" {
		msg := fmt.Sprintf("Invalid path: %s", rawPath)
		c.String(http.StatusBadRequest, msg)
		return
	}

	q := fmt.Sprintf("https://github.com%s.git", path)

	validURL := regexp.MustCompile(`^https://github\.com/[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+\.git$`)
	if !validURL.MatchString(q) {
		msg := fmt.Sprintf("Invalid URL: %s", q)
		c.String(http.StatusBadRequest, msg)
		return
	}

	repoDir := fmt.Sprintf("/tmp/sccs_%d", time.Now().UnixNano())
	if err := os.MkdirAll(repoDir, os.ModePerm); err != nil {
		msg := fmt.Sprintf("Failed to create directory: %s", err)
		c.String(http.StatusInternalServerError, msg)
		return
	}

	cmd := exec.Command("git", "clone", q, repoDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		msg := fmt.Sprintf("Failed to clone repository: %s, %s", err, output)
		c.String(http.StatusInternalServerError, msg)
		return
	}

	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			log.Printf("Failed to remove directory %s: %s", repoDir, err)
		}
	}()

	log.Printf("Repository cloned successfully to %s", repoDir)

	processor.DirFilePaths = []string{repoDir}
	processor.ConfigureGc()
	processor.ConfigureLazy(true)

	processor.Format = "tabular"
	tempFile, err := os.CreateTemp("", "sccs_output_*.txt")
	if err != nil {
		msg := fmt.Sprintf("Failed to create temp file: %s", err)
		c.String(http.StatusInternalServerError, msg)
		return
	}
	defer tempFile.Close()
	processor.FileOutput = tempFile.Name()
	processor.Process()

	c.File(tempFile.Name())
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func main() {
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/", func(c *gin.Context) {
		c.File("index.html")
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.NoRoute(scanHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
