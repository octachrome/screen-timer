namespace ScreenTimer.FullscreenHarness;

public sealed class HarnessForm : Form
{
    private readonly string _mode;
    private readonly int _resistCloseSeconds;
    private readonly DateTime _startTime = DateTime.Now;
    private readonly System.Windows.Forms.Timer _animTimer;

    private int _remainingResistSeconds;
    private int _closeAttempts;
    private float _hue;

    public HarnessForm(string mode, int resistCloseSeconds)
    {
        _mode = mode;
        _resistCloseSeconds = resistCloseSeconds;
        _remainingResistSeconds = resistCloseSeconds;

        Text = $"ScreenTimer FullscreenHarness — {mode}";
        DoubleBuffered = true;
        BackColor = Color.Black;

        switch (mode)
        {
            case "borderless":
                FormBorderStyle = FormBorderStyle.None;
                WindowState = FormWindowState.Maximized;
                break;

            case "exclusive":
                FormBorderStyle = FormBorderStyle.None;
                WindowState = FormWindowState.Maximized;
                TopMost = true;
                break;

            default: // windowed
                ClientSize = new Size(800, 600);
                StartPosition = FormStartPosition.CenterScreen;
                break;
        }

        _animTimer = new System.Windows.Forms.Timer { Interval = 50 };
        _animTimer.Tick += (_, _) =>
        {
            _hue = (_hue + 2f) % 360f;
            Invalidate();
        };
        _animTimer.Start();

        Log("Harness started");
    }

    protected override void OnPaint(PaintEventArgs e)
    {
        base.OnPaint(e);

        var g = e.Graphics;
        g.Clear(HslToColor(_hue, 0.8f, 0.4f));

        var elapsed = DateTime.Now - _startTime;
        var lines = new[]
        {
            $"Mode: {_mode}",
            $"Elapsed: {elapsed:hh\\:mm\\:ss\\.f}",
            $"Resist close: {(_remainingResistSeconds > 0 ? $"{_remainingResistSeconds}s remaining" : "OFF")}",
            $"Close attempts: {_closeAttempts}",
            "",
            "Press ESC or Alt+F4 to close"
        };

        using var font = new Font("Consolas", 20f, FontStyle.Bold);
        using var brush = new SolidBrush(Color.White);
        using var shadow = new SolidBrush(Color.FromArgb(80, 0, 0, 0));
        float y = ClientSize.Height / 2f - lines.Length * 30f;
        foreach (var line in lines)
        {
            var size = g.MeasureString(line, font);
            float x = (ClientSize.Width - size.Width) / 2f;
            g.DrawString(line, font, shadow, x + 2, y + 2);
            g.DrawString(line, font, brush, x, y);
            y += size.Height + 4;
        }
    }

    protected override void OnKeyDown(KeyEventArgs e)
    {
        if (e.KeyCode == Keys.Escape)
            Close();
        base.OnKeyDown(e);
    }

    protected override void OnFormClosing(FormClosingEventArgs e)
    {
        _closeAttempts++;
        Log($"Close event received (reason={e.CloseReason}, attempt={_closeAttempts})");

        if (_remainingResistSeconds > 0 && e.CloseReason != CloseReason.ApplicationExitCall)
        {
            Log($"Resisting close — {_remainingResistSeconds}s remaining");
            e.Cancel = true;
            _remainingResistSeconds--;
            Invalidate();
            return;
        }

        Log("Allowing close");
        _animTimer.Stop();
        base.OnFormClosing(e);
    }

    private static void Log(string message)
    {
        Console.WriteLine($"[{DateTime.Now:HH:mm:ss.fff}] {message}");
    }

    private static Color HslToColor(float h, float s, float l)
    {
        float c = (1f - Math.Abs(2f * l - 1f)) * s;
        float x = c * (1f - Math.Abs(h / 60f % 2f - 1f));
        float m = l - c / 2f;

        float r, g, b;
        if (h < 60) { r = c; g = x; b = 0; }
        else if (h < 120) { r = x; g = c; b = 0; }
        else if (h < 180) { r = 0; g = c; b = x; }
        else if (h < 240) { r = 0; g = x; b = c; }
        else if (h < 300) { r = x; g = 0; b = c; }
        else { r = c; g = 0; b = x; }

        return Color.FromArgb(
            (int)((r + m) * 255),
            (int)((g + m) * 255),
            (int)((b + m) * 255));
    }
}
