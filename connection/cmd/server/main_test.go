package main

import (
	"context"
	"testing"

	"connection/internal/event/codec"
	"connection/internal/gateway"
	kafkapb "connection/proto/kafka"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRegistry struct {
	removedRooms    []roomUser
	removedGateways []uint32
}

type roomUser struct {
	roomID uint32
	userID uint32
}

func (r *recordingRegistry) AddRoomUser(context.Context, uint32, uint32) error {
	return nil
}

func (r *recordingRegistry) RemoveRoomUser(_ context.Context, roomID, userID uint32) error {
	r.removedRooms = append(r.removedRooms, roomUser{roomID: roomID, userID: userID})
	return nil
}

func (r *recordingRegistry) SetUserGateway(context.Context, uint32, string) error {
	return nil
}

func (r *recordingRegistry) RemoveUserGateway(_ context.Context, userID uint32) error {
	r.removedGateways = append(r.removedGateways, userID)
	return nil
}

func TestDisconnectHandler_KeepsPresenceWhenAnotherConnectionForUserRemains(t *testing.T) {
	eventCodec := codec.NewJSONEventCodec[*kafkapb.KafkaEvent]()
	hub := newHub(eventCodec)
	reg := &recordingRegistry{}

	clientA := &gateway.Client{ID: 1, UserID: 42}
	clientB := &gateway.Client{ID: 2, UserID: 42}
	hub.AddClient(clientA)
	hub.AddClient(clientB)
	hub.AddClientToGroup(clientA.ID, 7)
	hub.AddClientToGroup(clientB.ID, 7)

	userID, groupIDs := hub.RemoveClientAndGroups(clientA.ID)
	require.Equal(t, uint32(42), userID)
	require.ElementsMatch(t, []uint32{7}, groupIDs)

	handler := newDisconnectHandler(hub, reg)
	handler(clientA.ID, userID, groupIDs)

	assert.Empty(t, reg.removedRooms)
	assert.Empty(t, reg.removedGateways)
}

func TestDisconnectHandler_RemovesOnlyStaleMembershipsAndLastGateway(t *testing.T) {
	eventCodec := codec.NewJSONEventCodec[*kafkapb.KafkaEvent]()
	hub := newHub(eventCodec)
	reg := &recordingRegistry{}

	clientA := &gateway.Client{ID: 1, UserID: 42}
	clientB := &gateway.Client{ID: 2, UserID: 42}
	hub.AddClient(clientA)
	hub.AddClient(clientB)
	hub.AddClientToGroup(clientA.ID, 7)
	hub.AddClientToGroup(clientA.ID, 8)
	hub.AddClientToGroup(clientB.ID, 8)

	userID, groupIDs := hub.RemoveClientAndGroups(clientA.ID)
	require.Equal(t, uint32(42), userID)
	require.ElementsMatch(t, []uint32{7, 8}, groupIDs)

	handler := newDisconnectHandler(hub, reg)
	handler(clientA.ID, userID, groupIDs)

	assert.Equal(t, []roomUser{{roomID: 7, userID: 42}}, reg.removedRooms)
	assert.Empty(t, reg.removedGateways)

	lastUserID, lastGroupIDs := hub.RemoveClientAndGroups(clientB.ID)
	require.Equal(t, uint32(42), lastUserID)
	require.ElementsMatch(t, []uint32{8}, lastGroupIDs)

	handler(clientB.ID, lastUserID, lastGroupIDs)

	assert.Equal(t, []roomUser{
		{roomID: 7, userID: 42},
		{roomID: 8, userID: 42},
	}, reg.removedRooms)
	assert.Equal(t, []uint32{42}, reg.removedGateways)
}
