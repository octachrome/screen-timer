using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Engine;

public static class AgentEngine
{
    private static readonly TimeSpan UsageFlushInterval = TimeSpan.FromSeconds(15);
    private static readonly TimeSpan ConfigPollInterval = TimeSpan.FromSeconds(30);

    public static EngineResult Tick(AgentState state, ForegroundSample sample, List<GroupRule>? newRules, DateTimeOffset? testPopupAt = null)
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

        // Check notifications and enforcement for all groups
        foreach (var rule in state.CurrentRules)
        {
            if (!state.GroupUsage.TryGetValue(rule.Name, out var groupState))
                continue;

            // Sum usage across all member processes
            var totalUsedSeconds = 0.0;
            foreach (var process in rule.Processes)
            {
                if (state.Apps.TryGetValue(process, out var appState))
                {
                    totalUsedSeconds += appState.UsedTodaySeconds;
                }
            }

            var dayOfWeek = sample.Timestamp.LocalDateTime.DayOfWeek;
            var isWeekend = (dayOfWeek == DayOfWeek.Saturday || dayOfWeek == DayOfWeek.Sunday);
            var activeBudgetMinutes = (isWeekend && rule.WeekendBudgetMinutes > 0) ? rule.WeekendBudgetMinutes : rule.DailyBudgetMinutes;
            var budgetSeconds = activeBudgetMinutes * 60.0;
            var remainingSeconds = budgetSeconds - totalUsedSeconds;

            // Check if current foreground exe is a member of this group
            var currentExeInGroup = currentTrackedExe is not null &&
                rule.Processes.Any(p => string.Equals(p, currentTrackedExe, StringComparison.OrdinalIgnoreCase));

            // Notification thresholds (fire only once each, only when a member process is in foreground)
            if (currentExeInGroup)
            {
                if (!groupState.Sent10Min && remainingSeconds <= 600 && remainingSeconds > 0)
                {
                    groupState.Sent10Min = true;
                    commands.Add(new ShowToastCommand(rule.Name, 10));
                }
                if (!groupState.Sent5Min && remainingSeconds <= 300 && remainingSeconds > 0)
                {
                    groupState.Sent5Min = true;
                    commands.Add(new ShowToastCommand(rule.Name, 5));
                }
                if (!groupState.Sent1Min && remainingSeconds <= 60 && remainingSeconds > 0)
                {
                    groupState.Sent1Min = true;
                    commands.Add(new ShowToastCommand(rule.Name, 1));
                }
            }

            // Enforcement: force-close when budget exhausted and a member is in foreground
            if (remainingSeconds <= 0)
            {
                groupState.Exhausted = true;
                if (currentExeInGroup)
                {
                    commands.Add(new ForceCloseCommand(currentTrackedExe!));
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
        }
        foreach (var group in state.GroupUsage.Values)
        {
            group.Sent10Min = false;
            group.Sent5Min = false;
            group.Sent1Min = false;
            group.Exhausted = false;
        }
    }

    private static void ApplyConfigRules(AgentState state, List<GroupRule> newRules)
    {
        state.CurrentRules = newRules;

        // Collect all process names referenced by any group
        var allProcesses = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var rule in newRules)
        {
            foreach (var process in rule.Processes)
            {
                allProcesses.Add(process);
            }
        }

        // Add AppUsageState for new processes
        foreach (var process in allProcesses)
        {
            if (!state.Apps.ContainsKey(process))
            {
                state.Apps[process] = new AppUsageState();
            }
        }

        // Remove processes no longer in any group
        var toRemove = state.Apps.Keys.Where(k => !allProcesses.Contains(k)).ToList();
        foreach (var key in toRemove)
        {
            state.Apps.Remove(key);
        }

        // Add GroupUsageState for new groups
        var groupNames = new HashSet<string>(newRules.Select(r => r.Name), StringComparer.OrdinalIgnoreCase);
        foreach (var name in groupNames)
        {
            if (!state.GroupUsage.ContainsKey(name))
            {
                state.GroupUsage[name] = new GroupUsageState();
            }
        }

        // Remove groups no longer in config
        var groupsToRemove = state.GroupUsage.Keys.Where(k => !groupNames.Contains(k)).ToList();
        foreach (var key in groupsToRemove)
        {
            state.GroupUsage.Remove(key);
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
