using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Engine;

public static class AgentEngine
{
    private static readonly TimeSpan UsageFlushInterval = TimeSpan.FromSeconds(15);
    private static readonly TimeSpan ConfigPollInterval = TimeSpan.FromSeconds(30);

    public static EngineResult Tick(AgentState state, ForegroundSample sample, List<AppRule>? newRules, DateTimeOffset? testPopupAt = null)
    {
        var commands = new List<EngineCommand>();
        var now = sample.Timestamp;
        var todayDate = now.LocalDateTime.Date.ToString("yyyy-MM-dd");

        // Midnight reset
        if (state.CurrentDate != todayDate)
        {
            ResetForNewDay(state, todayDate);
        }

        // Apply new config rules if provided
        if (newRules is not null)
        {
            ApplyConfigRules(state, newRules);
            state.LastConfigPollTime = now;
        }

        // Test popup
        if (testPopupAt.HasValue && (state.LastTestPopupTime == null || testPopupAt.Value > state.LastTestPopupTime.Value))
        {
            state.LastTestPopupTime = testPopupAt.Value;
            commands.Add(new ShowTestToastCommand());
        }

        // Track elapsed time attributed to the *previous* foreground exe
        if (state.LastTickTime != default)
        {
            var elapsed = (now - state.LastTickTime).TotalSeconds;
            if (elapsed > 0 && state.LastForegroundExe is not null)
            {
                var prevExe = state.LastForegroundExe;
                if (state.Apps.TryGetValue(prevExe, out var prevApp))
                {
                    prevApp.UsedTodaySeconds += elapsed;
                    prevApp.PendingUploadSeconds += elapsed;
                }
            }
        }

        // Update last sample info
        var currentTrackedExe = GetTrackedExeName(state, sample.ExeName);
        state.LastForegroundExe = currentTrackedExe;
        state.LastTickTime = now;

        // Check notifications and enforcement for all tracked apps
        foreach (var rule in state.CurrentRules)
        {
            if (!state.Apps.TryGetValue(rule.ExeName, out var appState))
                continue;

            var budgetSeconds = rule.DailyBudgetMinutes * 60.0;
            var remainingSeconds = budgetSeconds - appState.UsedTodaySeconds;

            // Notification thresholds (fire only once each, only for current foreground app)
            if (string.Equals(currentTrackedExe, rule.ExeName, StringComparison.OrdinalIgnoreCase))
            {
                if (!appState.Sent10Min && remainingSeconds <= 600 && remainingSeconds > 0)
                {
                    appState.Sent10Min = true;
                    commands.Add(new ShowToastCommand(rule.ExeName, 10));
                }
                if (!appState.Sent5Min && remainingSeconds <= 300 && remainingSeconds > 0)
                {
                    appState.Sent5Min = true;
                    commands.Add(new ShowToastCommand(rule.ExeName, 5));
                }
                if (!appState.Sent1Min && remainingSeconds <= 60 && remainingSeconds > 0)
                {
                    appState.Sent1Min = true;
                    commands.Add(new ShowToastCommand(rule.ExeName, 1));
                }
            }

            // Enforcement: force-close when budget exhausted and app is in foreground
            if (remainingSeconds <= 0)
            {
                appState.Exhausted = true;
                if (string.Equals(currentTrackedExe, rule.ExeName, StringComparison.OrdinalIgnoreCase))
                {
                    commands.Add(new ForceCloseCommand(rule.ExeName));
                }
            }
        }

        // Usage flush
        if ((now - state.LastUsageFlushTime).TotalSeconds >= UsageFlushInterval.TotalSeconds)
        {
            var usageReports = BuildUsageReports(state);
            if (usageReports.Count > 0)
            {
                commands.Add(new PushUsageCommand(new UsagePushDto { Usage = usageReports }));
                commands.Add(new PersistStateCommand());
            }
            state.LastUsageFlushTime = now;
        }

        return new EngineResult
        {
            UpdatedState = state,
            Commands = commands
        };
    }

    public static void MarkUsagePushSucceeded(AgentState state, UsagePushDto pushed)
    {
        foreach (var report in pushed.Usage)
        {
            if (state.Apps.TryGetValue(report.ExeName, out var appState))
            {
                appState.PendingUploadSeconds -= report.Seconds;
                if (appState.PendingUploadSeconds < 0)
                    appState.PendingUploadSeconds = 0;
            }
        }
    }

    private static void ResetForNewDay(AgentState state, string newDate)
    {
        state.CurrentDate = newDate;
        foreach (var app in state.Apps.Values)
        {
            app.UsedTodaySeconds = 0;
            app.Sent10Min = false;
            app.Sent5Min = false;
            app.Sent1Min = false;
            app.Exhausted = false;
        }
    }

    private static void ApplyConfigRules(AgentState state, List<AppRule> newRules)
    {
        state.CurrentRules = newRules;

        // Add AppUsageState for new apps
        foreach (var rule in newRules)
        {
            if (!state.Apps.ContainsKey(rule.ExeName))
            {
                state.Apps[rule.ExeName] = new AppUsageState();
            }
        }

        // Remove apps no longer in config
        var ruleSet = new HashSet<string>(newRules.Select(r => r.ExeName), StringComparer.OrdinalIgnoreCase);
        var toRemove = state.Apps.Keys.Where(k => !ruleSet.Contains(k)).ToList();
        foreach (var key in toRemove)
        {
            state.Apps.Remove(key);
        }
    }

    private static string? GetTrackedExeName(AgentState state, string? sampleExe)
    {
        if (sampleExe is null)
            return null;

        // Check if this exe is tracked (case-insensitive via dictionary comparer)
        if (state.Apps.ContainsKey(sampleExe))
            return sampleExe;

        return null;
    }

    private static List<UsageReportDto> BuildUsageReports(AgentState state)
    {
        var reports = new List<UsageReportDto>();
        foreach (var (exeName, appState) in state.Apps)
        {
            var seconds = (int)appState.PendingUploadSeconds;
            if (seconds > 0)
            {
                reports.Add(new UsageReportDto { ExeName = exeName, Seconds = seconds, TotalSeconds = (int)appState.UsedTodaySeconds });
            }
        }
        return reports;
    }
}
