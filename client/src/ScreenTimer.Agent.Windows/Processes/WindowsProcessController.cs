using System.Diagnostics;
using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Processes;

public sealed class WindowsProcessController : IProcessController
{
    public async Task ForceCloseAsync(string exeName)
    {
        foreach (var process in Process.GetProcessesByName(exeName))
        {
            try
            {
                process.CloseMainWindow();
                if (!process.WaitForExit(5000))
                {
                    process.Kill();
                }
            }
            catch
            {
                // Swallow exceptions so a single kill failure doesn't crash the agent.
            }
            finally
            {
                process.Dispose();
            }
        }

        await Task.CompletedTask;
    }
}
