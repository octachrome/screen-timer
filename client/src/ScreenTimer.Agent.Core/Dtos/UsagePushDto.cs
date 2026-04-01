using System.Text.Json.Serialization;

namespace ScreenTimer.Agent.Core.Dtos;

public sealed class UsagePushDto
{
    [JsonPropertyName("usage")]
    public List<UsageReportDto> Usage { get; set; } = new();
}
