package main

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
)

// GetGroups godoc
//
//	@Summary		Get groups known by cloud-init
//	@Description	Get meta-data and cloud-init config for all groups known to
//	@Description	cloud-init.  Note that group membership is managed outside of
//	@Description	the cloud-init service, normally in SMD.
//	@Tags			admin,groups
//	@Produce		json
//	@Success		200	{object}	map[string]cistore.ClusterDefaults
//	@Failure		500	{object}	nil
//	@Router			/admin/groups [get]
func (h CiHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	var (
		groups map[string]cistore.GroupData
		bytes  []byte
		err    error
	)
	groups = h.store.GetGroups()
	bytes, err = json.MarshalIndent(groups, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// AddGroupHandler godoc
//
//	@Summary		Add a new group
//	@Description	Add a new group to cloud-init corresponding to an SMD group.
//	@Description	Group-wide meta-data and/or a cloud-init configuration (in
//	@Description	either plain or base64 encoding) can be specified.
//	@Description
//	@Description	If successful, a 201 Created status is returned and the
//	@Description	`Location` header is set to the new group's groups endpoint,
//	@Description	`/groups/{name}`.
//	@Description
//	@Description	If request parsing fails, a 422 Unprocessable Entity status is
//	@Description	returned. If adding group data to the data store fails, a 409
//	@Description	Conflict status is returned.
//	@Tags			admin,groups
//	@Accept			json
//	@Success		201		{object}	nil
//	@Failure		409		{object}	nil
//	@Failure		422		{object}	nil
//	@Header			201		{string}	Location			"/groups/{id}"
//	@Param			group	body		cistore.GroupData	true	"Group data"
//	@Router			/admin/groups [post]
func (h CiHandler) AddGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		data cistore.GroupData
		err  error
	)

	data, err = parseData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = h.store.AddGroupData(data.Name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Location", "/groups/"+data.Name)
	w.WriteHeader(http.StatusCreated)

}

// GetGroupHandler godoc
//
//	@Summary		Get data for single group
//	@Description	Get meta-data and cloud-init config for a single group known to
//	@Description	cloud-init.
//	@Tags			admin,groups
//	@Produce		json
//	@Success		200	{object}	cistore.GroupData
//	@Failure		404	{object}	nil
//	@Failure		500	{object}	nil
//	@Param			id	path		string	true	"Group ID"
//	@Router			/admin/groups/{id} [get]
func (h CiHandler) GetGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		id    string
		data  cistore.GroupData
		bytes []byte
		err   error
	)
	id = chi.URLParam(r, "id")

	data, err = h.store.GetGroupData(id)
	if err != nil {
		if reflect.DeepEqual(data, cistore.GroupData{}) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	bytes, err = json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateGroupHandler godoc
//
//	@Summary		Set group-specific meta-data and/or cloud-init config
//	@Description	Set meta-data or cloud-init configuration for a specific group,
//	@Description	overwriting any previous values.
//	@Description
//	@Description	If successful, a 201 Created status is returned and the
//	@Description	`Location` header is set to the new group's groups endpoint,
//	@Description	`/groups/{group}`. This operation is idempotent and replaces
//	@Description	any existing content.
//	@Tags			admin,groups
//	@Accept			json
//	@Success		201			{object}	nil
//	@Failure		422			{object}	nil
//	@Failure		500			{object}	nil
//	@Header			201			{string}	Location			"/groups/{name}"
//	@Param			name		path		string				true	"Group name"
//	@Param			group_data	body		cistore.GroupData	true	"Group data"
//	@Router			/admin/groups/{name} [put]
func (h CiHandler) UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		groupName string
		data      cistore.GroupData
		err       error
	)

	groupName = chi.URLParam(r, "name")

	data, err = parseData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// update group key-value data
	err = h.store.UpdateGroupData(groupName, data, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/groups/"+data.Name)
	w.WriteHeader(http.StatusCreated)
}

// RemoveGroupHandler godoc
//
//	@Summary		Delete a group
//	@Description	Delete a group with its meta-data and cloud-init config.
//	@Tags			admin,groups
//	@Success		200	{object}	nil
//	@Failure		500	{object}	nil
//	@Param			id	path		string	true	"Group ID"
//	@Router			/admin/groups/{id} [delete]
func (h CiHandler) RemoveGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		id  string
		err error
	)
	id = chi.URLParam(r, "id")
	err = h.store.RemoveGroupData(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
