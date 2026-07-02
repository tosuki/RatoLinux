using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using System.Text.RegularExpressions;
using System.Globalization;

namespace RatinhoDesktop.Models;

/// <summary>
/// Tipo de som sintetizado a ser usado para o efeito de "clique" de cada bichinho.
/// Cada um gera uma forma de onda diferente em SoundGenerator, então não precisamos
/// de arquivos de áudio externos para cada bicho.
/// </summary>
public enum SoundCharacter
{
    Squeak,   // rato: guincho agudo e curto
    Moo,      // vaca: som grave e longo, com vibrato
    Meow,     // gato: miado de duas sílabas
    Pop,      // efeitos "genéricos" (dança/limpeza/etc): som curto e alegre
    Chime     // som tipo "sininho", usado para o eisque
}

/// <summary>
/// Descreve um bichinho selecionável: nome de exibição, gif, o som associado a ele e o BPM base.
/// </summary>
public class PetDefinition
{
    public string Id { get; }
    public string DisplayName { get; set; }
    public string PackUri { get; }
    public SoundCharacter Sound { get; set; }
    public double BaseBpm { get; set; }

    public PetDefinition(string id, string displayName, string packUri, SoundCharacter sound, double baseBpm = 120.0)
    {
        Id = id;
        DisplayName = displayName;
        PackUri = packUri;
        Sound = sound;
        BaseBpm = baseBpm;
    }

    /// <summary>
    /// Catálogo com todos os bichinhos disponíveis. É populado inicialmente com os embutidos
    /// e depois incrementado dinamicamente escaneando a pasta Assets.
    /// </summary>
    public static readonly List<PetDefinition> Catalog = new();

    static PetDefinition()
    {
        // 1. Adiciona os padrões embutidos
        Catalog.Add(new PetDefinition("rato", "Ratinho", "pack://application:,,,/Assets/rato.gif", SoundCharacter.Squeak, 120.0));
        Catalog.Add(new PetDefinition("vaca", "Vaca", "pack://application:,,,/Assets/Novos/vaca.gif", SoundCharacter.Moo, 80.0));
        Catalog.Add(new PetDefinition("cat", "Gatinho", "pack://application:,,,/Assets/Novos/cat.gif", SoundCharacter.Meow, 110.0));
        Catalog.Add(new PetDefinition("silly-cat-dance", "Gato Dançarino", "pack://application:,,,/Assets/Novos/silly-cat-dance.gif", SoundCharacter.Pop, 130.0));
        Catalog.Add(new PetDefinition("dancing-dance", "Dançarino", "pack://application:,,,/Assets/Novos/dancing-dance.gif", SoundCharacter.Pop, 125.0));
        Catalog.Add(new PetDefinition("limpando", "Limpando", "pack://application:,,,/Assets/Novos/limpando.gif", SoundCharacter.Pop, 100.0));
        Catalog.Add(new PetDefinition("eisque", "Eisque", "pack://application:,,,/Assets/Novos/eisque.gif", SoundCharacter.Chime, 120.0));

        // 2. Escaneia a pasta local de Assets por novos GIFs e carrega configurações personalizadas se existirem
        try
        {
            string assetsPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "Assets");
            if (Directory.Exists(assetsPath))
            {
                var configMap = new Dictionary<string, PetConfig>();
                string configPath = Path.Combine(assetsPath, "pets.json");
                if (File.Exists(configPath))
                {
                    try
                    {
                        string json = File.ReadAllText(configPath);
                        var wrapper = JsonSerializer.Deserialize<PetConfigWrapper>(json, new JsonSerializerOptions
                        {
                            PropertyNameCaseInsensitive = true
                        });
                        if (wrapper?.Pets != null)
                        {
                            foreach (var config in wrapper.Pets)
                            {
                                if (!string.IsNullOrEmpty(config.Id))
                                {
                                    configMap[config.Id.ToLower()] = config;
                                }
                            }
                        }
                    }
                    catch
                    {
                        // Se falhar ao ler o JSON, prossegue sem os overrides
                    }
                }

                // Aplica overrides aos pets embutidos que foram encontrados no JSON
                foreach (var pet in Catalog)
                {
                    if (configMap.TryGetValue(pet.Id, out var config))
                    {
                        pet.DisplayName = config.DisplayName ?? pet.DisplayName;
                        pet.BaseBpm = config.BaseBpm ?? pet.BaseBpm;
                        if (Enum.TryParse<SoundCharacter>(config.Sound, true, out var soundVal))
                        {
                            pet.Sound = soundVal;
                        }
                    }
                }

                // Escaneia a pasta por arquivos GIFs físicos
                var gifFiles = Directory.GetFiles(assetsPath, "*.gif", SearchOption.AllDirectories);
                foreach (var file in gifFiles)
                {
                    string filename = Path.GetFileNameWithoutExtension(file);
                    string id = filename.ToLower();

                    // Se já estiver no catálogo (como embutido), não duplica
                    if (Catalog.Exists(p => p.Id == id))
                        continue;

                    // Formata um nome amigável de exibição (ex: silly-cat-dance -> Silly Cat Dance)
                    string displayName = Regex.Replace(filename, @"[-_]+", " ");
                    displayName = CultureInfo.CurrentCulture.TextInfo.ToTitleCase(displayName);

                    SoundCharacter sound = SoundCharacter.Pop;
                    if (id.Contains("vaca") || id.Contains("cow")) sound = SoundCharacter.Moo;
                    else if (id.Contains("cat") || id.Contains("gato")) sound = SoundCharacter.Meow;
                    else if (id.Contains("rato") || id.Contains("mouse")) sound = SoundCharacter.Squeak;
                    else if (id.Contains("eisque")) sound = SoundCharacter.Chime;

                    double baseBpm = 120.0;

                    // Aplica configurações do pets.json se existirem para este novo pet
                    if (configMap.TryGetValue(id, out var c))
                    {
                        displayName = c.DisplayName ?? displayName;
                        baseBpm = c.BaseBpm ?? baseBpm;
                        if (Enum.TryParse<SoundCharacter>(c.Sound, true, out var soundVal))
                        {
                            sound = soundVal;
                        }
                    }

                    string fileUri = new Uri(file).AbsoluteUri;
                    Catalog.Add(new PetDefinition(id, displayName, fileUri, sound, baseBpm));
                }
            }
        }
        catch
        {
            // Se falhar no escaneamento, usa apenas a lista padrão
        }
    }

    public static PetDefinition GetByIdOrDefault(string? id)
    {
        if (!string.IsNullOrEmpty(id))
        {
            foreach (var pet in Catalog)
            {
                if (pet.Id == id) return pet;
            }
        }
        return Catalog[0];
    }

    private class PetConfigWrapper
    {
        public List<PetConfig>? Pets { get; set; }
    }

    private class PetConfig
    {
        public string? Id { get; set; }
        public string? DisplayName { get; set; }
        public double? BaseBpm { get; set; }
        public string? Sound { get; set; }
    }
}
