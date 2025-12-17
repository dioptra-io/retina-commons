package api

import (
	"net"
	"time"
)

type IPVersion uint8

const (
	TypeIPv4 IPVersion = 4
	TypeIPv6 IPVersion = 6
)

type Protocol uint8

const (
	ICMP   Protocol = 1
	UDP    Protocol = 17
	ICMPv6 Protocol = 58
)

type ICMPNextHeader struct {
	FirstHalfWord  uint16 `json:"first_half_word"`
	SecondHalfWord uint16 `json:"second_half_word"`
}

type UDPNextHeader struct {
	SourcePort      uint16 `json:"source_port"`
	DestinationPort uint16 `json:"destination_port"`
}

type ICMPv6NextHeader struct {
	FirstHalfWord  uint16 `json:"first_half_word"`
	SecondHalfWord uint16 `json:"second_half_word"`
}

type NextHeader struct {
	ICMPNextHeader   *ICMPNextHeader   `json:"icmp_next_header"`
	UDPNextHeader    *UDPNextHeader    `json:"udp_next_header"`
	ICMPv6NextHeader *ICMPv6NextHeader `json:"icmpv6_next_header"`
}

type ProbingDirective struct {
	IPVersion          IPVersion  `json:"ip_version"`
	Protocol           Protocol   `json:"protocol"`
	AgentID            string     `json:"agent_id"`
	DestinationAddress net.IP     `json:"destination_address"`
	NearTTL            uint8      `json:"near_ttl"`
	NextHeader         NextHeader `json:"next_header"`
}

type Agent struct {
	AgentID string `json:"agent_id"`
}

type Info struct {
	TTL               uint8     `json:"ttl"`
	ReplyAddress      net.IP    `json:"reply_address"`
	PayloadSize       uint8     `json:"payload_size"`
	SentTimestamp     time.Time `json:"sent_timestamp"`
	ReceivedTimestamp time.Time `json:"received_timestamp"`
}

type ForwardingInfoElement struct {
	Agent               Agent     `json:"agent"`
	IPVersion           IPVersion `json:"ip_version"`
	Protocol            Protocol  `json:"protocol"`
	SourceAddress       net.IP    `json:"source_address"`
	DestinationAddress  net.IP    `json:"destination_address"`
	NearInfo            Info      `json:"near_info"`
	FarInfo             Info      `json:"far_info"`
	ProductionTimestamp time.Time `json:"production_timestamp"`
}
