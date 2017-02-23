package demoinfocs

import (
	"github.com/gogo/protobuf/proto"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"sync"
)

var packetEntitiesPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(msg.CSVCMsg_PacketEntities)
	},
}

var gameEventPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(msg.CSVCMsg_GameEvent)
	},
}

var tickPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(msg.CNETMsg_Tick)
	},
}

type byteSliceBacker struct {
	slice []byte
}

var byteSliceBackerPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &byteSliceBacker{slice: make([]byte, 0, 256)}
	},
}

func (p *Parser) parsePacket() {
	for !p.bitreader.ChunkFinished() {
		cmd := int(p.bitreader.ReadVarInt32())
		size := int(p.bitreader.ReadVarInt32())

		p.bitreader.BeginChunk(size << 3)
		var m proto.Message
		switch cmd {
		case int(msg.SVC_Messages_svc_PacketEntities):
			m = packetEntitiesPool.Get().(*msg.CSVCMsg_PacketEntities)
			defer packetEntitiesPool.Put(m)

		case int(msg.SVC_Messages_svc_GameEventList):
			m = new(msg.CSVCMsg_GameEventList)

		case int(msg.SVC_Messages_svc_GameEvent):
			m = gameEventPool.Get().(*msg.CSVCMsg_GameEvent)
			defer gameEventPool.Put(m)

		case int(msg.SVC_Messages_svc_CreateStringTable):
			m = new(msg.CSVCMsg_CreateStringTable)

		case int(msg.SVC_Messages_svc_UpdateStringTable):
			m = new(msg.CSVCMsg_UpdateStringTable)

		case int(msg.NET_Messages_net_Tick):
			m = tickPool.Get().(*msg.CNETMsg_Tick)
			defer tickPool.Put(m)

		case int(msg.SVC_Messages_svc_UserMessage):
			m = new(msg.CSVCMsg_UserMessage)

		default:
			// We don't care about anything else for now
			p.bitreader.EndChunk()
			continue
		}

		backer := byteSliceBackerPool.Get().(*byteSliceBacker)
		p.bitreader.ReadBytesInto(&backer.slice, size)

		proto.Unmarshal(backer.slice, m)
		p.msgQueue <- m

		// Reset to 0 length and pool
		backer.slice = backer.slice[:0]
		byteSliceBackerPool.Put(backer)

		p.bitreader.EndChunk()
	}

	// Make sure the created events are consumed so they can be pooled
	p.msgDispatcher.SyncQueues(p.msgQueue)
}
