package main

import (
	"fmt"
	"net/http"
	yaml "gopkg.in/yaml.v2"
	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"github.com/travisbcotton/harbor/internal/memstore"
	"github.com/travisbcotton/harbor/pkg/citypes"
	"github.com/travisbcotton/harbor/internal/smdclient"
)

func main() {
	router := gin.Default()

	store := memstore.NewMemStore()
	sm := smdclient.NewSMDClient("http://ochami-vm:27779")
	ciHandler := NewCiHandler(store, sm)

	router.GET("/harbor", ciHandler.ListEntries)
	router.POST("/harbor", ciHandler.AddEntry)
	router.GET("/harbor/:id", ciHandler.GetEntry)
	router.GET("/harbor/:id/user-data",ciHandler.GetUserData)
	router.GET("/harbor/:id/meta-data",ciHandler.GetMetaData)
	router.GET("/harbor/:id/vendor-data",ciHandler.GetVendorData)
	router.PUT("/harbor/:id", ciHandler.UpdateEntry)
	router.DELETE("harbor/:id", ciHandler.DeleteEntry)
	

	router.Run()
}


type CiHandler struct {
	store ciStore
	sm *smdclient.SMDClient
}

func NewCiHandler(s ciStore, c *smdclient.SMDClient) *CiHandler {
	return &CiHandler{
		store: s,
		sm: c,
	}
}

type ciStore interface {
	Add(name string, ci citypes.CI) error
	Get(name string, sm *smdclient.SMDClient) (citypes.CI, error)
	List() (map[string]citypes.CI, error)
	Update(name string, ci citypes.CI) error
	Remove(name string) error
}

func (h CiHandler) ListEntries(c *gin.Context) {
	ci, err := h.store.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(200, ci)
}

func (h CiHandler) AddEntry(c *gin.Context) {
	var ci citypes.CI
	if err := c.ShouldBindJSON(&ci); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := slug.Make(ci.Name)

	err := h.store.Add(id, ci)
        if err != nil {
                if err == memstore.ExistingEntryErr {
                        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
                        return
                }
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }

	c.JSON(http.StatusOK, ci.Name)
}

func (h CiHandler) GetEntry(c *gin.Context) {
	id := c.Param("id")

	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}

	c.JSON(200, ci)
}

func (h CiHandler) GetUserData(c *gin.Context) {
	id := c.Param("id")

	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}
	ud,err := yaml.Marshal(ci.CIData.UserData)
	if err != nil {
		fmt.Print(err)
	}
	s := fmt.Sprintf("#cloud-config\n%s", string(ud[:]))
	//c.Header("Content-Type", "text/yaml")
	c.Data(200,"text/yaml", []byte(s))
}

func (h CiHandler) GetMetaData(c *gin.Context) {
        id := c.Param("id")

        ci, err := h.store.Get(id, h.sm)
        if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        }

        c.YAML(200, ci.CIData.MetaData)
}

func (h CiHandler) GetVendorData(c *gin.Context) {
        id := c.Param("id")

        ci, err := h.store.Get(id, h.sm)
        if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        }

        c.YAML(200, ci.CIData.VendorData)
}

func (h CiHandler) UpdateEntry(c *gin.Context) {
	var ci citypes.CI
	if err := c.ShouldBindJSON(&ci); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	err := h.store.Update(id, ci)
	if err != nil {
		if err == memstore.NotFoundErr {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, id)
}


func (h CiHandler) DeleteEntry(c *gin.Context) {
	id := c.Param("id")

	err := h.store.Remove(id)
	if err != nil {
		if err == memstore.NotFoundErr {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
