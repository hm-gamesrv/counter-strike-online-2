package adminapi

import (
	"net/http"
	"strconv"
	"time"

	. "github.com/KouKouChan/CSO2-Server/blademaster/typestruct"
	. "github.com/KouKouChan/CSO2-Server/config"
	. "github.com/KouKouChan/CSO2-Server/servermanager"
)

type statusResponse struct {
	Version        string `json:"version"`
	UptimeSeconds  int64  `json:"uptimeSeconds"`
	OnlineUsers    int    `json:"onlineUsers"`
	GamePort       uint32 `json:"gamePort"`
	HolepunchPort  uint32 `json:"holepunchPort"`
	AdminPort      uint32 `json:"adminPort"`
	DataDir        string `json:"dataDir"`
	UserDataDir    string `json:"userDataDir"`
	Accounts       int    `json:"accounts"`
	MaxUsers       uint32 `json:"maxUsers"`
	ShopEnabled    bool   `json:"shopEnabled"`
	AllWeaponsFree bool   `json:"unlockAllWeapons"`
}

func (s *adminServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{
		Version:        ServerVersion,
		UptimeSeconds:  int64(time.Since(s.startedAt).Seconds()),
		OnlineUsers:    OnlineUserCount(),
		GamePort:       Env.TCPPort,
		HolepunchPort:  Env.UDPPort,
		AdminPort:      Env.AdminPort,
		DataDir:        DataDir,
		UserDataDir:    UserDataDir,
		Accounts:       AccountCount(),
		MaxUsers:       MaxUsers,
		ShopEnabled:    Conf.EnableShop,
		AllWeaponsFree: Conf.UnlockAllWeapons,
	})
}

func (s *adminServer) handleListPlayers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"count":   OnlineUserCount(),
		"players": ListOnlineUsers(),
	})
}

func (s *adminServer) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := ListAllAccounts()
	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(accounts),
		"accounts": accounts,
	})
}

func (s *adminServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	detail, err := CreateAccount(req)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

func (s *adminServer) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	detail, err := GetAccount(r.PathValue("name"))
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *adminServer) handleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	var patch AccountPatch
	if err := decodeBody(w, r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	detail, err := UpdateAccount(r.PathValue("name"), patch)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *adminServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := DeleteAccount(name); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "deleted", "username": name})
}

func (s *adminServer) handleKick(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := KickUser(name); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "kicked", "username": name})
}

type addItemRequest struct {
	ItemID uint32 `json:"itemId"`
	Count  uint16 `json:"count"`
}

func (s *adminServer) handleAddItem(w http.ResponseWriter, r *http.Request) {
	var req addItemRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ItemID == 0 {
		writeError(w, http.StatusBadRequest, "itemId must be a positive integer")
		return
	}
	if req.Count == 0 {
		req.Count = 1
	}

	name := r.PathValue("name")
	if err := AddItem(name, req.ItemID, req.Count); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "added", "username": name, "itemId": req.ItemID, "count": req.Count,
	})
}

func (s *adminServer) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.ParseUint(r.PathValue("itemId"), 10, 32)
	if err != nil || itemID == 0 {
		writeError(w, http.StatusBadRequest, "itemId must be a positive integer")
		return
	}

	name := r.PathValue("name")
	if err = RemoveItem(name, uint32(itemID)); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "removed", "username": name, "itemId": itemID,
	})
}

func (s *adminServer) handleSetVIP(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := SetVIP(name); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "vip", "username": name})
}

func (s *adminServer) handleSetGM(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := SetGM(name); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "gm", "username": name})
}

func (s *adminServer) handleDeleteRoom(w http.ResponseWriter, r *http.Request) {
	channelServer, err := parseByte(r.PathValue("channelServer"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "channelServer must be an integer between 1 and 255")
		return
	}
	channel, err := parseByte(r.PathValue("channel"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "channel must be an integer between 1 and 255")
		return
	}
	room, err := strconv.ParseUint(r.PathValue("room"), 10, 16)
	if err != nil || room == 0 {
		writeError(w, http.StatusBadRequest, "room must be an integer between 1 and 65535")
		return
	}

	if err = DeleteRoom(channelServer, channel, uint16(room)); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "closed", "channelServerIndex": channelServer, "channelIndex": channel, "roomId": room,
	})
}

func (s *adminServer) handleSave(w http.ResponseWriter, r *http.Request) {
	if err := SaveAll(); err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "saved"})
}
