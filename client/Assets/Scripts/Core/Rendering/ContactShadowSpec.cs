namespace ThinhThan.Core.Rendering
{
    // Contact-shadow ellipse dimensions per actor size profile: the width
    // follows the profile's collider reference width (reference px at the
    // 100 PPU gameplay import, i.e. px / 100 = world meters; SPIRIT_BEAST
    // has no collider and uses its 48 px silhouette) and the height is 30%
    // of the width for the flat ground ellipse.
    public readonly struct ContactShadowSpec
    {
        public ContactShadowSpec(float widthMeters, float heightMeters, float opacity)
        {
            WidthMeters = widthMeters;
            HeightMeters = heightMeters;
            Opacity = opacity;
        }

        public float WidthMeters { get; }
        public float HeightMeters { get; }
        public float Opacity { get; }

        public static ContactShadowSpec For(ActorSizeProfile profile)
        {
            switch (profile)
            {
                case ActorSizeProfile.MonsterSmall:
                    return new ContactShadowSpec(0.30f, 0.09f, 0.45f);
                case ActorSizeProfile.MonsterMedium:
                    return new ContactShadowSpec(0.50f, 0.15f, 0.45f);
                case ActorSizeProfile.MonsterElite:
                    return new ContactShadowSpec(0.80f, 0.24f, 0.45f);
                case ActorSizeProfile.BossLarge:
                    return new ContactShadowSpec(1.20f, 0.36f, 0.45f);
                case ActorSizeProfile.WorldBoss:
                    return new ContactShadowSpec(1.50f, 0.45f, 0.45f);
                case ActorSizeProfile.SpiritBeast:
                    return new ContactShadowSpec(0.48f, 0.14f, 0.45f);
                case ActorSizeProfile.Character:
                default:
                    return new ContactShadowSpec(0.40f, 0.12f, 0.45f);
            }
        }
    }
}
