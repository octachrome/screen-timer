using System.Text.Json.Serialization;

namespace ScreenTimer.Agent.Core.Dtos;

public sealed class AgentConfigResponseDto
{
    [JsonPropertyName("apps")]
    public List<AppConfigDto> Apps { get; set; } = new();

    [JsonPropertyName("test_popup_at")]
    public string? TestPopupAt { get; set; }
}
