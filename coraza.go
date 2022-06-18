package coraza

import (
	"io"
	"log"
	"net/http"

	"github.com/corazawaf/coraza/v2"
	"github.com/gin-gonic/gin"
)

func Coraza(waf *coraza.Waf) gin.HandlerFunc {
	return func(c *gin.Context) {
		tx := waf.NewTransaction()
		defer tx.ProcessLogging()
		if it, err := tx.ProcessRequest(c.Request); err != nil {
			renderError(c, "Coraza: Failed to process request")
			return
		} else if it != nil {
			forbidden(c, tx)
			return
		}
		oldwriter := c.Writer
		c.Writer = &responseWriter{
			tx:             tx,
			ResponseWriter: oldwriter,
		}
		c.Next()
		if it, err := tx.ProcessResponseBody(); err != nil {
			renderError(c, "Coraza: Failed to process response body")
		} else if it != nil {
			forbidden(c, tx)
		}
		// we dump the body to the writer
		reader, err := tx.ResponseBodyBuffer.Reader()
		if err != nil {
			renderError(c, "Coraza: Failed to get response body reader")
			return
		}
		io.Copy(oldwriter, reader)
	}
}

func renderError(c *gin.Context, content string) {
	c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "msg": content})
}

func forbidden(c *gin.Context, tx *coraza.Transaction) {
	log.Printf("Coraza blocked request from %s with %s, RuleID %d \n", c.RemoteIP(), c.Request.URL, tx.Interruption.RuleID)
	c.JSON(http.StatusForbidden, gin.H{"status": "interrupted", "transaction": tx.ID})
}
