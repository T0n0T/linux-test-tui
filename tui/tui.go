package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.bug.st/serial"
)

var (
	mainStyle = lipgloss.NewStyle().
		Margin(1, 2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
)

type SerialModel struct {
	port        string
	baudRate    int
	portConn    serial.Port
	bytesSent   int
	bytesRecv   int
	packetsSent int
	packetsRecv int
	packetsLost int
	packetsErr  int
	isQuitting  bool
	maxPackets  int
}

type serialTestMsg struct{}
type serialTestResult struct{}

func calculateHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func NewSerialModel(port string, baudRate int, count int) (*SerialModel, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}

	portConn, err := serial.Open(port, mode)
	if err != nil {
		return &SerialModel{}, fmt.Errorf("failed to open serial port: %w", err)
	}

	return &SerialModel{
		port:       port,
		baudRate:   baudRate,
		portConn:   portConn,
		maxPackets: count,
	}, nil
}

func (m *SerialModel) Init() tea.Cmd {
	return m.serialTest()
}

func (m *SerialModel) serialTest() tea.Cmd {
	return func() tea.Msg {
		// 生成测试数据
		testData := fmt.Sprintf("Test packet %d", m.packetsSent+1)
		// 发送数据
		n, err := m.portConn.Write([]byte(testData))
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}
		m.bytesSent += n
		m.packetsSent++
		sentHash := calculateHash(testData)

		// 接收数据
		buf := make([]byte, 256)
		n, err = m.portConn.Read(buf)
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}
		m.bytesRecv += n
		m.packetsRecv++
		recvData := string(buf[:n])
		recvHash := calculateHash(recvData)

		if sentHash != recvHash {
			m.packetsErr++
		}
		if m.packetsSent > m.packetsRecv {
			m.packetsLost = m.packetsSent - m.packetsRecv
		}

		// 检查是否完成测试
		if m.packetsSent >= m.maxPackets {
			m.isQuitting = true
			return serialTestResult{}
		}

		// 继续发送下一个数据包
		return serialTestMsg{}
	}
}

func (m *SerialModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.isQuitting = true
			if m.portConn != nil {
				// 忽略关闭错误
				_ = m.portConn.Close()
			}
			return m, tea.Quit
		}
	case serialTestMsg:
		return m, m.serialTest()

	case serialTestResult:
		return m, tea.Quit

	case error:
		// if !m.isQuitting {
		// 	log.Printf("Serial error: %v", msg)
		// }
		return m, nil
	default:
		return m, nil
	}
	return m, nil
}

func (m *SerialModel) View() string {
	title := fmt.Sprintf("Serial Loopback Test - %s @ %d baud", m.port, m.baudRate)

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

	if m.isQuitting {
		stats += "\n"
	}

	return mainStyle.Render(title + stats)
}
