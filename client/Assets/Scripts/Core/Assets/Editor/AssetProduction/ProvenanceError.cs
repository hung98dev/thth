namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // One provenance validation failure. Path identifies the offending
    // file_path (or register field) so errors are always path-specific.
    public sealed class ProvenanceError
    {
        public readonly string Path;
        public readonly string Message;

        public ProvenanceError(string path, string message)
        {
            Path = path;
            Message = message;
        }

        public override string ToString()
        {
            return Path + ": " + Message;
        }
    }
}
