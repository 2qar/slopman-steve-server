package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGuildPlayerGet(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	slot := parseInt64(formValue(r, "slot"))
	g, err := s.guilds.GetForPlayer(r.Context(), acc.ID, slot)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	writeText(w, http.StatusOK, fmt.Sprintf("%d|%s|%s|%s|%d|%d",
		g.ID, g.Name, g.Description, g.Tag, g.Color, g.Status))
}

func (s *Server) handleAccountGuildsGet(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	guilds, err := s.guilds.ListForAccount(r.Context(), acc.ID)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	if len(guilds) == 0 {
		writeText(w, http.StatusOK, "0")
		return
	}

	fields := make([]string, 0, len(guilds))
	for _, g := range guilds {
		fields = append(fields, fmt.Sprintf("%d|%d|%s", g.GuildID, g.Slot, g.Name))
	}
	writeText(w, http.StatusOK, strings.Join(fields, "$"))
}

func (s *Server) handleGuildAdd(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0|Invalid session")
		return
	}

	name := formValue(r, "name")
	tag := formValue(r, "tag")
	description := formValue(r, "description")
	color := parseInt64(formValue(r, "color"))
	slot := parseInt64(formValue(r, "slot"))

	guildID, err := s.guilds.CreateGuild(r.Context(), name, tag, description, color, acc.ID, slot)
	if err != nil {
		writeText(w, http.StatusOK, "0|Guild creation failed")
		return
	}
	writeText(w, http.StatusOK, fmt.Sprintf("%d|", guildID))
}

func (s *Server) handleGuildDelete(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	guildID := parseInt64(formValue(r, "guild_id"))
	if err := s.guilds.DeleteGuild(r.Context(), guildID, acc.ID); err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleGuildEdit(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	guildID := parseInt64(formValue(r, "guild_id"))
	description := formValue(r, "description")
	if err := s.guilds.EditDescription(r.Context(), guildID, acc.ID, description); err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleGuildGetAllPlayers(w http.ResponseWriter, r *http.Request) {
	_, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	guildID := parseInt64(formValue(r, "guild_id"))
	page := parseInt64(formValue(r, "page"))
	members, err := s.guilds.Members(r.Context(), guildID, page)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	fields := []string{strconv.Itoa(len(members))}
	for _, mem := range members {
		fields = append(fields, fmt.Sprintf("%s|%d|%d|%d", mem.Name, mem.Class, mem.AccountID, mem.Slot))
	}
	writeText(w, http.StatusOK, strings.Join(fields, "§"))
}

func (s *Server) handleGuildPlayerAdd(w http.ResponseWriter, r *http.Request) {
	caller, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	guildID := parseInt64(formValue(r, "guild_id"))
	targetAccountID := parseInt64(formValue(r, "target_account_id"))
	targetSlot := parseInt64(formValue(r, "target_slot"))
	status := parseInt64(formValue(r, "status"))

	if guildID == 0 || targetAccountID == 0 || targetSlot == 0 {
		writeText(w, http.StatusOK, "0")
		return
	}

	callerStatus, member, err := s.guilds.HighestStatus(r.Context(), guildID, caller.ID)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	if !member {
		writeText(w, http.StatusOK, "0")
		return
	}

	switch status {
	case 1:
		// Only an owner may promote a member to owner.
		if callerStatus != 1 {
			writeText(w, http.StatusOK, "0")
			return
		}
		isMember, err := s.guilds.IsMemberInSlot(r.Context(), guildID, targetAccountID, targetSlot)
		if err != nil {
			writeText(w, http.StatusOK, "0")
			return
		}
		if !isMember {
			writeText(w, http.StatusOK, "0")
			return
		}
	case -1:
		// Only an owner may remove members.
		if callerStatus != 1 {
			writeText(w, http.StatusOK, "0")
			return
		}
	default:
		// Status 0 is an invite or a self-join for another character slot.
		if status != 0 {
			writeText(w, http.StatusOK, "0")
			return
		}
		if targetAccountID == caller.ID {
			// Self-join is only valid if this account already belongs to the
			// guild on another slot.
			alreadyMember, err := s.guilds.IsMember(r.Context(), guildID, caller.ID)
			if err != nil {
				writeText(w, http.StatusOK, "0")
				return
			}
			if !alreadyMember {
				writeText(w, http.StatusOK, "0")
				return
			}
		} else if callerStatus != 1 {
			// Inviting another account requires owner status.
			writeText(w, http.StatusOK, "0")
			return
		}
	}

	if status == -1 {
		if err := s.guilds.RemoveMember(r.Context(), guildID, targetAccountID, targetSlot); err != nil {
			writeText(w, http.StatusOK, "0")
			return
		}
	} else {
		if err := s.guilds.SetMember(r.Context(), guildID, targetAccountID, targetSlot, status); err != nil {
			writeText(w, http.StatusOK, "0")
			return
		}
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleGuildPlayerDelete(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	guildID := parseInt64(formValue(r, "guild_id"))
	slot := parseInt64(formValue(r, "slot"))
	if err := s.guilds.RemoveMember(r.Context(), guildID, acc.ID, slot); err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, "1")
}
