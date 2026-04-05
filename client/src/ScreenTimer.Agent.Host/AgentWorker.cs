using Microsoft.Extensions.Logging;
using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Interfaces;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Host;

public sealed class AgentWorker : BackgroundService
{
    private readonly IForegroundWindowProbe _probe;
    private readonly IAgentApiClient _apiClient;
    private readonly INotificationSink _notifications;
    private readonly IProcessController _processController;
    private readonly IStateStore _stateStore;
    private readonly IClock _clock;
    private readonly ILogger<AgentWorker> _logger;

    private readonly TimeSpan _tickInterval = TimeSpan.FromSeconds(1);
    private readonly TimeSpan _configPollInterval = TimeSpan.FromSeconds(30);
    private static readonly TimeSpan MaxBackoff = TimeSpan.FromMinutes(1);

    private AgentState _state = new();
    private int _configFailures;
    private int _usagePushFailures;
    private string? _previousForegroundExe;

    public AgentWorker(
        IForegroundWindowProbe probe,
        IAgentApiClient apiClient,
        INotificationSink notifications,
        IProcessController processController,
        IStateStore stateStore,
        IClock clock,
        ILogger<AgentWorker> logger)
    {
        _probe = probe;
        _apiClient = apiClient;
        _notifications = notifications;
        _processController = processController;
        _stateStore = stateStore;
        _clock = clock;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("Agent worker starting");

        var loaded = _stateStore.Load();
        if (loaded is not null)
        {
            _state = loaded;
            _logger.LogInformation("Restored persisted state for date {Date}", _state.CurrentDate);
        }

        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await TickAsync(stoppingToken);
            }
            catch (Exception ex) when (ex is not OperationCanceledException)
            {
                _logger.LogError(ex, "Error during tick");
            }

            await Task.Delay(_tickInterval, stoppingToken);
        }
    }

    private async Task TickAsync(CancellationToken ct)
    {
        var sample = _probe.Sample();
        var now = _clock.Now;

        // Log foreground changes
        if (!string.Equals(sample.ExeName, _previousForegroundExe, StringComparison.OrdinalIgnoreCase))
        {
            _logger.LogInformation("Foreground changed: {PreviousExe} -> {CurrentExe}",
                _previousForegroundExe ?? "(none)", sample.ExeName ?? "(none)");
            _previousForegroundExe = sample.ExeName;
        }

        // Poll config if due (with backoff on failures)
        List<AppRule>? newRules = null;
        DateTimeOffset? testPopupAt = null;
        var configInterval = GetBackoffInterval(_configPollInterval, _configFailures);
        if ((now - _state.LastConfigPollTime).TotalSeconds >= configInterval.TotalSeconds)
        {
            try
            {
                var configResponse = await _apiClient.GetConfigAsync(ct);
                newRules = configResponse.Apps.Select(c => new AppRule
                {
                    ExeName = c.ExeName,
                    DailyBudgetMinutes = c.DailyBudgetMinutes
                }).ToList();
                if (DateTimeOffset.TryParse(configResponse.TestPopupAt, out var parsed))
                    testPopupAt = parsed;
                _configFailures = 0;
                _logger.LogDebug("Config polled: {Count} app(s)", newRules.Count);
            }
            catch (Exception ex) when (ex is not OperationCanceledException)
            {
                _configFailures++;
                _logger.LogWarning(ex, "Config poll failed (attempt {Attempt}, next retry in {Backoff}s)",
                    _configFailures, GetBackoffInterval(_configPollInterval, _configFailures).TotalSeconds);
                _state.LastConfigPollTime = now;
            }
        }

        var result = AgentEngine.Tick(_state, sample, newRules, testPopupAt);
        _state = result.UpdatedState;

        foreach (var command in result.Commands)
        {
            await DispatchCommandAsync(command, ct);
        }
    }

    private async Task DispatchCommandAsync(EngineCommand command, CancellationToken ct)
    {
        switch (command)
        {
            case ShowToastCommand toast:
                _logger.LogInformation("Toast: {ExeName} — {Minutes} min remaining", toast.ExeName, toast.RemainingMinutes);
                _notifications.ShowToast(toast.ExeName, toast.RemainingMinutes);
                break;

            case PushUsageCommand push:
                try
                {
                    await _apiClient.PushUsageAsync(push.Payload, ct);
                    AgentEngine.MarkUsagePushSucceeded(_state, push.Payload);
                    _usagePushFailures = 0;
                    _logger.LogDebug("Usage pushed: {Count} app(s)", push.Payload.Usage.Count);
                }
                catch (Exception ex) when (ex is not OperationCanceledException)
                {
                    _usagePushFailures++;
                    _logger.LogWarning(ex, "Usage push failed (attempt {Attempt}), will retry",
                        _usagePushFailures);
                }
                break;

            case ForceCloseCommand close:
                _logger.LogWarning("Enforcing close: {ExeName}", close.ExeName);
                try
                {
                    await _processController.ForceCloseAsync(close.ExeName);
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Failed to force-close {ExeName}", close.ExeName);
                }
                break;

            case PersistStateCommand:
                try
                {
                    _stateStore.Save(_state);
                    _logger.LogDebug("State persisted");
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Failed to persist state");
                }
                break;

            case ShowTestToastCommand:
                _logger.LogInformation("Test popup requested");
                _notifications.ShowToast("Test", 0);
                break;
        }
    }

    private static TimeSpan GetBackoffInterval(TimeSpan baseInterval, int failureCount)
    {
        if (failureCount <= 0)
            return baseInterval;

        var backoffSeconds = baseInterval.TotalSeconds * Math.Pow(2, Math.Min(failureCount, 10));
        return TimeSpan.FromSeconds(Math.Min(backoffSeconds, MaxBackoff.TotalSeconds));
    }
}
