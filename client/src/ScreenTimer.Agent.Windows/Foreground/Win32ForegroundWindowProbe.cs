using System.Diagnostics;
using System.Runtime.InteropServices;
using ScreenTimer.Agent.Core.Interfaces;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Windows.Foreground;

public sealed class Win32ForegroundWindowProbe(IClock clock) : IForegroundWindowProbe
{
    [DllImport("user32.dll")]
    private static extern IntPtr GetForegroundWindow();

    [DllImport("user32.dll", SetLastError = true)]
    private static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);

    public ForegroundSample Sample()
    {
        var exeName = GetForegroundProcessName();
        return new ForegroundSample(exeName, clock.Now);
    }

    private static string? GetForegroundProcessName()
    {
        var hwnd = GetForegroundWindow();
        if (hwnd == IntPtr.Zero)
            return null;

        GetWindowThreadProcessId(hwnd, out var pid);
        if (pid == 0)
            return null;

        try
        {
            using var process = Process.GetProcessById((int)pid);
            return process.ProcessName;
        }
        catch
        {
            return null;
        }
    }
}
