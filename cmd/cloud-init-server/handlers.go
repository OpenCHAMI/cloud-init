package main

import (
	"fmt"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	yaml "gopkg.in/yaml.v2"
)

type CiHandler struct {
	store ciStore
	sm    *smdclient.SMDClient
}

func NewCiHandler(s ciStore, c *smdclient.SMDClient) *CiHandler {
	return &CiHandler{
		store: s,
		sm:    c,
	}
}

// ListEntries godoc
// @Summary List all cloud-init entries
// @Description List all cloud-init entries
// @Produce json
// @Success 200 {object} map[string]CI
// @Router /harbor [get]
func (h CiHandler) ListEntries(c *gin.Context) {
	ci, err := h.store.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(200, ci)
}

// AddEntry godoc
// @Summary Add a new cloud-init entry
// @Description Add a new cloud-init entry
// @Accept json
// @Produce json
// @Param ci body CI true "Cloud-init entry to add"
// @Success 200 {string} string "name of the new entry"
// @Failure 400 {string} string "bad request"
// @Failure 500 {string} string "internal server error"
// @Router /harbor [post]
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

// GetEntry godoc
// @Summary Get a cloud-init entry
// @Description Get a cloud-init entry
// @Produce json
// @Param id path string true "ID of the cloud-init entry to get"
// @Success 200 {object} CI
// @Failure 404 {string} string "not found"
// @Router /harbor/{id} [get]
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
	ud, err := yaml.Marshal(ci.CIData.UserData)
	if err != nil {
		fmt.Print(err)
	}
	s := fmt.Sprintf("#cloud-config\n%s", string(ud[:]))
	//c.Header("Content-Type", "text/yaml")
	c.Data(200, "text/yaml", []byte(s))
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
