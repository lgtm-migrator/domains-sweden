package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Index renders the HTML of the index page
func (controller Controller) Index(c *gin.Context) {
	pd := controller.DefaultPageData(c)
	pd.Title = pd.Trans("Home")
	c.HTML(http.StatusOK, "index.html", pd)
}
