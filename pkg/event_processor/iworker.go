package event_processor

import (
	"ecapture/user"
	"go.uber.org/zap"
	"log"
	"time"
)

type IWorker interface {

	// 定时器1 ，定时判断没有后续包，则解析输出

	// 定时器2， 定时判断没后续包，则通知上层销毁自己

	// 收包
	Write(event user.IEventStruct) error
	GetUUID() string
}

const (
	MAX_TICKER_COUNT = 5  // 定时器超时时间
	MAX_CHAN_LEN     = 16 // 包队列长度
	//MAX_EVENT_LEN    = 16 // 事件数组长度
)

type eventWorker struct {
	incoming chan user.IEventStruct
	//events      []user.IEventStruct
	status      PROCESS_STATUS
	packetType  PACKET_TYPE
	ticker      *time.Ticker
	tickerCount uint8
	UUID        string
	processor   *EventProcessor
	parser      IParser
}

func NewEventWorker(uuid string, processor *EventProcessor) IWorker {
	eWorker := &eventWorker{}
	eWorker.init(uuid, processor)
	go func() {
		eWorker.Run()
	}()
	return eWorker
}

func (this *eventWorker) init(uuid string, processor *EventProcessor) {
	this.ticker = time.NewTicker(time.Second * 1)
	this.incoming = make(chan user.IEventStruct, MAX_CHAN_LEN)
	this.UUID = uuid
	this.processor = processor
}

func (this *eventWorker) GetUUID() string {
	return this.UUID
}

func (this *eventWorker) Write(event user.IEventStruct) error {
	this.incoming <- event
	return nil
}

// 输出包内容
func (this *eventWorker) Display() {
	if this.parser == nil || !this.parser.IsDone() {
		return
	}

	if this.parser.ParserType() != PARSER_TYPE_HTTP_REQUEST {
		//TODO 临时i调试
		return
	}
	log.Println("eventWorker:", this.UUID, "display")

	//  输出包内容
	b := this.parser.Display()
	this.processor.GetLogger().Info("eventWorker:display packet", zap.String("uuid", this.UUID), zap.Int("payload", len(b)))
	// 重置状态
	this.parser.Reset()

	// 设定状态、重置包类型
	this.status = PROCESS_STATE_DONE
	this.packetType = PACKET_TYPE_NULL

}

// 解析类型，输出
func (this *eventWorker) parserEvent(event user.IEventStruct) {
	if this.status == PROCESS_STATE_INIT {
		// 识别包类型
		parser := NewParser(event.Payload())
		this.parser = parser
	}
	_, err := this.parser.Write(event.Payload()[:event.PayloadLen()])
	if err != nil {
		this.processor.GetLogger().Fatal("eventWorker: detect packet type error:", zap.String("uuid", this.UUID), zap.Error(err))
	}

	//this.processor.GetLogger().Info("eventWorker:detect packet type", zap.String("uuid", this.UUID), zap.Uint8("type", uint8(this.parser.ParserType())), zap.String("IParser Name", this.parser.Name()))
	// 是否接收完成，能否输出
	if this.parser.IsDone() {
		this.Display()
	}
}

func (this *eventWorker) Run() {
	for {
		select {
		case _ = <-this.ticker.C:
			// 输出包
			if this.tickerCount > MAX_TICKER_COUNT {
				this.Close()
				return
			}
			this.tickerCount++
		case event := <-this.incoming:
			// reset tickerCount
			this.tickerCount = 0
			this.parserEvent(event)
		}
	}

}

func (this *eventWorker) Close() {
	this.Display()
	this.tickerCount = 0
	this.processor.delWorkerByUUID(this)
}
