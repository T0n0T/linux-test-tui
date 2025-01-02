package tui

import (
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.bug.st/serial"
)

type SerialModel struct {
	port        string
	baudRate    int
	sentData    []string
	recvData    []string
	portConn    serial.Port
	bytesSent   int
	bytesRecv   int
	packetsSent int
	packetsRecv int
}

func NewSerialModel(port string, baudRate int) (SerialModel, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}

	portConn, err := serial.Open(port, mode)
	if err != nil {
		return SerialModel{}, fmt.Errorf("failed to open serial port: %w", err)
	}

	return SerialModel{
		port:     port,
		baudRate: baudRate,
		portConn: portConn,
	}, nil
}

func (m SerialModel) Init() tea.Cmd {
	return tea.Batch(
		m.listenForSerialData(),
		m.sendTestData(),
	)
}

func (m SerialModel) listenForSerialData() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 128)
		for {
			n, err := m.portConn.Read(buf)
			if err != nil {
				return err
			}
			if n > 0 {
				return serialDataReceived{
					data:    string(buf[:n]),
					bytes:   n,
					packets: 1,
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (m SerialModel) sendTestData() tea.Cmd {
	return func() tea.Msg {
		testData := "Test message\n"
		time.Sleep(100 * time.Millisecond) // 增加发送间隔
		n, err := m.portConn.Write([]byte(testData))
		if err != nil {
			return err
		}
		return serialDataSent{
			data:    testData,
			bytes:   n,
			packets: 1,
		}
	}
}

func (m SerialModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.portConn.Close()
			fmt.Printf("\nTest Summary:\n"+
				"Bytes Sent: %d\n"+
				"Bytes Received: %d\n"+
				"Packets Sent: %d\n"+
				"Packets Received: %d\n",
				m.bytesSent, m.bytesRecv, m.packetsSent, m.packetsRecv)
			if m.packetsSent > 0 {
				lossRate := float64(m.packetsSent-m.packetsRecv) / float64(m.packetsSent) * 100
				fmt.Printf("Packet Loss: %.2f%%\n", lossRate)
			}
			return m, tea.Quit
		}

	case serialDataSent:
		m.sentData = append(m.sentData, msg.data)
		m.bytesSent += msg.bytes
		m.packetsSent += msg.packets
		return m, m.sendTestData()

	case serialDataReceived:
		m.recvData = append(m.recvData, msg.data)
		m.bytesRecv += msg.bytes
		m.packetsRecv += msg.packets
		return m, m.listenForSerialData()

	case error:
		log.Printf("Serial error: %v", msg)
		return m, nil
	}
	return m, nil
}

func (m SerialModel) View() string {
	title := fmt.Sprintf("Serial Loopback Test - %s @ %d baud", m.port, m.baudRate)

	sent := "Sent Data:\n"
	for _, data := range m.sentData {
		sent += fmt.Sprintf("> %s", data)
	}

	stats := fmt.Sprintf("\nStatistics:\n"+
		"Bytes Sent: %d\n"+
		"Bytes Received: %d\n"+
		"Packets Sent: %d\n"+
		"Packets Received: %d\n",
		m.bytesSent, m.bytesRecv, m.packetsSent, m.packetsRecv)

	if m.packetsSent > 0 {
		lossRate := float64(m.packetsSent-m.packetsRecv) / float64(m.packetsSent) * 100
		stats += fmt.Sprintf("Packet Loss: %.2f%%\n", lossRate)
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(1, 2)

	return style.Render(title + stats)
}

type serialDataSent struct {
	data    string
	bytes   int
	packets int
}

type serialDataReceived struct {
	data    string
	bytes   int
	packets int
}
