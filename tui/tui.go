package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
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
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
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
	spinner     spinner.Model
	progress    progress.Model
	statsTable  table.Model
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

	s := spinner.New()
	s.Spinner = spinner.Monkey
	s.Style = spinnerStyle

	// 初始化进度条
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	// 初始化表格
	columns := []table.Column{
		{Title: "Metric", Width: 20},
		{Title: "Value", Width: 20},
	}

	rows := []table.Row{
		{"Bytes Sent", "0"},
		{"Bytes Received", "0"},
		{"Packets Sent", "0"},
		{"Packets Received", "0"},
		{"Packet Loss", "0%"},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)
	// 设置默认焦点行为Packet Loss行（第4行）
	t.SetCursor(4)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)

	return &SerialModel{
		port:       port,
		baudRate:   baudRate,
		portConn:   portConn,
		maxPackets: count,
		spinner:    s,
		progress:   prog,
		statsTable: t,
	}, nil
}

func (m *SerialModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.serialTest(),
	)
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case error:
		if !m.isQuitting {
			log.Printf("Serial error: %v", msg)
		}
		return m, nil
	default:
		return m, nil
	}
	return m, nil
}

func (m *SerialModel) View() string {
	title := m.spinner.View() + fmt.Sprintf(" Serial Loopback Test - %s @ %d baud", m.port, m.baudRate)

	// 计算进度
	progressPercent := float64(m.packetsSent) / float64(m.maxPackets)
	if progressPercent > 1.0 {
		progressPercent = 1.0
	}

	// 更新表格数据
	lossRate := 0.0
	if m.packetsSent > 0 {
		lossRate = float64(m.packetsSent-m.packetsRecv) / float64(m.packetsSent) * 100
	}

	m.statsTable.SetRows([]table.Row{
		{"Bytes Sent", fmt.Sprintf("%d", m.bytesSent)},
		{"Bytes Received", fmt.Sprintf("%d", m.bytesRecv)},
		{"Packets Sent", fmt.Sprintf("%d", m.packetsSent)},
		{"Packets Received", fmt.Sprintf("%d", m.packetsRecv)},
		{"Packet Loss", fmt.Sprintf("%.2f%%", lossRate)},
	})

	// 构建视图
	progressView := fmt.Sprintf("\n\n %s \n", m.progress.ViewAs(progressPercent))
	statsView := fmt.Sprintf("\n%s\n", m.statsTable.View())

	if m.isQuitting {
		statsView += "\n"
	}

	return mainStyle.Render(title + progressView + statsView)
}
