namespace ThinhThan.Core.Assets
{
    // Catalog ID prefix set used by catalog-backed asset keys
    // (client_assets.md § Stable Asset Keys). The first segment of a
    // catalog ID declares its kind; zone/effect rows carry no
    // presentation facet themselves and only drive grouping.
    public enum CatalogAssetKind
    {
        Class,
        Monster,
        Boss,
        Npc,
        Beast,
        Skill,
        Item,
        Cosmetic,
        Map,
        Dungeon,
        Instance,
        Effect,
        Zone,
    }
}
