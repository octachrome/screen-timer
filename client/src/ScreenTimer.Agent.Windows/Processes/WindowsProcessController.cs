using System.Diagnostics;
using Microsoft.Extensions.Logging;
using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Processes;

public sealed class WindowsProcessController : IProcessController
{
    private readonly ILogger<WindowsProcessController> _logger;

    public WindowsProcessController(ILogger<WindowsProcessController> logger)
    {
        _logger = logger;
    }

    public async Task ForceCloseAsync(string exeName)
    {
        // Process.GetProcessesByName expects the name without extension
        var processName = Path.GetFileNameWithoutExtension(exeName);
        var processes = Process.GetProcessesByName(processName);

        if (processes.Length == 0)
        {
            _logger.LogWarning("ForceClose: no processes found matching '{ProcessName}'", processName);
            return;
        }

        _logger.LogInformation("ForceClose: found {Count} process(es) matching '{ProcessName}'",
            processes.Length, processName);

        foreach (var process in processes)
        {
            try
            {
                _logger.LogInformation(
                    "ForceClose: attempting to close PID {Pid}, ProcessName='{Name}', HasExited={HasExited}, MainWindowHandle={Handle}",
                    process.Id, process.ProcessName, process.HasExited, process.MainWindowHandle);

                var closeResult = process.CloseMainWindow();
                _logger.LogInformation(
                    "ForceClose: CloseMainWindow returned {Result} for PID {Pid}",
                    closeResult, process.Id);

                using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(5));
                await process.WaitForExitAsync(cts.Token);
                _logger.LogInformation("ForceClose: PID {Pid} exited gracefully", process.Id);
            }
            catch (OperationCanceledException)
            {
                _logger.LogWarning(
                    "ForceClose: graceful close timed out for PID {Pid}, attempting Kill()",
                    process.Id);
                try
                {
                    process.Kill();
                    _logger.LogInformation("ForceClose: Kill() succeeded for PID {Pid}", process.Id);
                }
                catch (Exception killEx)
                {
                    _logger.LogError(killEx,
                        "ForceClose: Kill() failed for PID {Pid}", process.Id);
                }
            }
            catch (Exception ex)
            {
                _logger.LogError(ex,
                    "ForceClose: unexpected error for PID {Pid}", process.Id);
            }
            finally
            {
                process.Dispose();
            }
        }
    }
}
