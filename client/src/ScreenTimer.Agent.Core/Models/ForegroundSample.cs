namespace ScreenTimer.Agent.Core.Models;

public sealed record ForegroundSample(string? ExeName, DateTimeOffset Timestamp);
