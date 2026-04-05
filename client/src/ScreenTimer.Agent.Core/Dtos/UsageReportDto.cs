using System.Text.Json.Serialization;

namespace ScreenTimer.Agent.Core.Dtos;

public sealed class UsageReportDto
{
    [JsonPropertyName("exe_name")]
    public string ExeName { get; set; } = "";

    [JsonPropertyName("seconds")]
    public int Seconds { get; set; }

    [JsonPropertyName("total_seconds")]
    public int TotalSeconds { get; set; }
}
