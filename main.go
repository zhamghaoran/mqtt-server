package main

import (
	"bytes"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"leetcode/constant"
	"leetcode/handler"
	. "leetcode/packet"
	"log"
	"net"
)

func main() {
	// 创建一个tcp服务
	tcpAddr, _ := net.ResolveTCPAddr("tcp", ":1883")
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("MQTT server listening on localhost:1883")

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		log.Println("New client connected:", conn.RemoteAddr())
		packet, err := ReadPacket(conn)
		if err != nil {
			break
		}
		log.Println(packet.String())
		// 处理数据
		typeCode, err := handleDeclaredStruct(packet)
		if err != nil {
			return
		}
		sendACK(conn, typeCode)
	}
	log.Println("Client disconnected:", conn.RemoteAddr())
}
func handleDeclaredStruct(packet ControlPacket) (int, error) {
	// 获取到方法列表
	handlers := handler.GetHandler()
	// 将方法列表传入
	typeCode, err := ExecuteHandler(packet, handlers)
	if err != nil {
		return typeCode, err
	}
	return typeCode, nil
}
func ExecuteHandler(packet ControlPacket, handler handler.HandlerI) (int, error) {
	typeCode := packet.Type()
	var err error
	switch typeCode {
	case 1:
		return packet.(*ConnectPacket).Type(), handler.ConnectHandle(packet.(*ConnectPacket))
	case 2:
		packet = packet.(*ConnackPacket)
		return packet.Type(), nil
	case 3:
		packet = packet.(*PublishPacket)
		return packet.Type(), nil
	case 4:
		packet = packet.(*PubackPacket)
		return packet.Type(), nil
	case 5:
		packet = packet.(*PubrecPacket)
		return packet.Type(), nil
	case 6:
		packet = packet.(*PubrelPacket)
		return packet.Type(), nil
	case 7:
		packet = packet.(*PubcompPacket)
		return packet.Type(), nil
	case 8:
		packet = packet.(*SubscribePacket)
		return packet.Type(), nil
	case 9:
		packet = packet.(*SubackPacket)
		return packet.Type(), nil
	case 10:
		packet = packet.(*UnsubscribePacket)
		return packet.Type(), nil
	case 11:
		packet = packet.(*UnsubackPacket)
		return packet.Type(), nil
	case 12:
		packet = packet.(*PingreqPacket)
		return packet.Type(), nil
	case 13:
		packet = packet.(*PingrespPacket)
		return packet.Type(), nil
	case 14:
		packet = packet.(*DisconnectPacket)
		return packet.Type(), nil
	default:
		err = fmt.Errorf("unsupported packet type : %d", typeCode)
		return 0, err
	}
	return 0, nil
}
func sendACK(conn net.Conn, messageType int) {
	var err error
	var i bytes.Buffer
	switch messageType {
	case constant.CONNECT:
		i.Reset()
		connackPacket := packets.NewControlPacket(Connack).(*packets.ConnackPacket)
		_ = connackPacket.Write(&i)
		fmt.Println(i.Bytes())
		_, _ = conn.Write(i.Bytes())
		//_, err = conn.Write([]byte{0x20, byte(constant.CONNACK), 0x00, 0x00})
	case constant.PINGREG:
		i.Reset()
		pingrespPacket := packets.NewControlPacket(Pingresp).(*packets.PingrespPacket)
		_ = pingrespPacket.Write(&i)
		fmt.Println(i.Bytes())
		_, _ = conn.Write(i.Bytes())
		//_, err = conn.Write([]byte{byte(constant.PINGRESQ), 0x00})
	case constant.SUBSCRIBE:
		i.Reset()
		subackpacket := packets.NewControlPacket(Suback).(*packets.SubackPacket)
		_ = subackpacket.Write(&i)
		fmt.Println(i.Bytes())
		_, _ = conn.Write(i.Bytes())
	}
	if err != nil {
		log.Printf("sendACK err : %s", err.Error())
	}
}

type MQTTHeader struct {
	MessageType byte
	Dup         bool
	QosLevel    byte
	Retain      bool
	Remaining   int
}

func (header *MQTTHeader) ParseHeader(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("invalid header length")
	}

	header.MessageType = data[0] >> 4
	header.Dup = ((data[0] >> 3) & 0x01) == 1
	header.QosLevel = (data[0] >> 1) & 0x03
	header.Retain = (data[0] & 0x01) == 1

	var multiplier uint32 = 1
	var value uint32 = 0
	var pos int = 1
	var b byte

	for {
		if pos >= len(data) {
			return fmt.Errorf("invalid header length")
		}

		b = data[pos]
		value += uint32(b&0x7F) * multiplier
		multiplier *= 128

		if multiplier > 128*128*128 {
			return fmt.Errorf("invalid header length")
		}
		pos++
		if b&0x80 == 0 {
			break
		}
	}
	header.Remaining = int(value)

	return nil
}
