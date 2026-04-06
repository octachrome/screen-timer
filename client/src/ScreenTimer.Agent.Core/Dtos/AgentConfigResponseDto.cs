using System.Text.Json.Serialization;

namespace ScreenTimer.Agent.Core.Dtos;

public sealed class AgentConfigResponseDto
{
    [JsonPropertyName("groups")]
    public List<GroupConfigDto> Groups { get; set; } = new();

    [JsonPropertyName("test_popup_at")]
    public string? TestPopupAt { get; set; }
}
