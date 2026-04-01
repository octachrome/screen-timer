using System.Text.Json;
using ScreenTimer.Agent.Core.Interfaces;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Windows.Storage;

public sealed class JsonStateStore : IStateStore
{
    private static readonly JsonSerializerOptions SerializerOptions = new()
    {
        WriteIndented = true
    };

    private readonly string _filePath;

    public JsonStateStore(string filePath)
    {
        _filePath = filePath;
    }

    public static string DefaultFilePath =>
        Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "ScreenTimer",
            "agent-state.json");

    public AgentState? Load()
    {
        if (!File.Exists(_filePath))
            return null;

        var json = File.ReadAllText(_filePath);
        return JsonSerializer.Deserialize<AgentState>(json, SerializerOptions);
    }

    public void Save(AgentState state)
    {
        var dir = Path.GetDirectoryName(_filePath)!;
        Directory.CreateDirectory(dir);

        var tmpPath = _filePath + ".tmp";
        var json = JsonSerializer.Serialize(state, SerializerOptions);
        File.WriteAllText(tmpPath, json);
        File.Move(tmpPath, _filePath, overwrite: true);
    }
}
