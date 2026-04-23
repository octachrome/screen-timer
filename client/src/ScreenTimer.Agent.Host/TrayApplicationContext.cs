using System.Diagnostics;
using System.Reflection;

namespace ScreenTimer.Agent.Host;

public sealed class TrayApplicationContext : ApplicationContext
{
    private readonly NotifyIcon _notifyIcon;
    private readonly string _logFilePath;

    public TrayApplicationContext(string logFilePath)
    {
        _logFilePath = logFilePath;

        var menu = new ContextMenuStrip();
        menu.Items.Add("Open Logs", null, OnOpenLogs);
        menu.Items.Add("Exit", null, OnExit);

        _notifyIcon = new NotifyIcon
        {
            Icon = LoadEmbeddedIcon(),
            Text = "Screen Timer",
            Visible = true,
            ContextMenuStrip = menu
        };
    }

    private static Icon LoadEmbeddedIcon()
    {
        var assembly = Assembly.GetExecutingAssembly();
        var stream = assembly.GetManifestResourceStream(
            "ScreenTimer.Agent.Host.Resources.tray.ico");
        return stream is not null ? new Icon(stream) : SystemIcons.Application;
    }

    private void OnOpenLogs(object? sender, EventArgs e)
    {
        Process.Start("notepad.exe", _logFilePath);
    }

    private void OnExit(object? sender, EventArgs e)
    {
        Application.Exit();
    }

    protected override void Dispose(bool disposing)
    {
        if (disposing)
        {
            _notifyIcon.Visible = false;
            _notifyIcon.Dispose();
        }
        base.Dispose(disposing);
    }
}
