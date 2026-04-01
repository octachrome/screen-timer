using System.Diagnostics;
using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Processes;

public sealed class WindowsProcessController : IProcessController
{
    public async Task ForceCloseAsync(string exeName)
    {
        // Process.GetProcessesByName expects the name without extension
        var processName = Path.GetFileNameWithoutExtension(exeName);

        foreach (var process in Process.GetProcessesByName(processName))
        {
            try
            {
                process.CloseMainWindow();
                using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(5));
                await process.WaitForExitAsync(cts.Token);
            }
            catch (OperationCanceledException)
            {
                // Graceful close timed out — kill the process
                try
                {
                    process.Kill();
                }
                catch
                {
                    // Swallow kill failure so the agent continues running
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
    }
}
