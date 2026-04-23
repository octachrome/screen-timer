using ScreenTimer.Agent.Core.Interfaces;
using ScreenTimer.Agent.Host;
using ScreenTimer.Agent.Windows;
using ScreenTimer.Agent.Windows.Foreground;
using ScreenTimer.Agent.Windows.Http;
using ScreenTimer.Agent.Windows.Notifications;
using ScreenTimer.Agent.Windows.Processes;
using ScreenTimer.Agent.Windows.Storage;
using Serilog;

var logDirectory = Path.Combine(
    Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
    "ScreenTimer", "logs");
Directory.CreateDirectory(logDirectory);
var logFilePath = Path.Combine(logDirectory, "agent.log");

Log.Logger = new LoggerConfiguration()
    .MinimumLevel.Information()
    .WriteTo.File(logFilePath,
        fileSizeLimitBytes: 10 * 1024 * 1024,
        rollOnFileSizeLimit: true,
        retainedFileCountLimit: 5,
        outputTemplate: "{Timestamp:yyyy-MM-dd HH:mm:ss.fff} [{Level:u3}] {Message:lj}{NewLine}{Exception}")
    .CreateLogger();

try
{
    var builder = Host.CreateApplicationBuilder(args);
    builder.Logging.ClearProviders();
    builder.Services.AddSerilog();

    var serverUrl = builder.Configuration.GetValue<string>("ServerUrl") ?? "http://localhost:8080";

    builder.Services.AddSingleton<IClock, SystemClock>();
    builder.Services.AddSingleton<IForegroundWindowProbe, Win32ForegroundWindowProbe>();
    builder.Services.AddSingleton<INotificationSink, ToastNotificationSink>();
    builder.Services.AddSingleton<IProcessController, WindowsProcessController>();
    builder.Services.AddSingleton<IStateStore>(_ => new JsonStateStore(JsonStateStore.DefaultFilePath));
    builder.Services.AddHttpClient<IAgentApiClient, AgentApiClient>(client =>
    {
        client.BaseAddress = new Uri(serverUrl);
    });
    builder.Services.AddHostedService<AgentWorker>();

    var host = builder.Build();

    Log.Information("Screen Timer Agent starting — server: {ServerUrl}", serverUrl);

    await host.StartAsync();

    Application.EnableVisualStyles();
    Application.SetCompatibleTextRenderingDefault(false);
    Application.Run(new TrayApplicationContext(logFilePath));

    await host.StopAsync();
}
catch (Exception ex)
{
    Log.Fatal(ex, "Agent terminated unexpectedly");
}
finally
{
    Log.CloseAndFlush();
}
