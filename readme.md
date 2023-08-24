
# codepage-unzip: A tool to unzip non-unicode ZIP archives with proper codepage handling

Standard Unix/Linux **unzip** does not go well with old, non-unicode ZIPs.
This tool properly creates unicode filenames from such old ZIP files.


## Install
With a go commandline,
```
go install github.com/mixcode/codepage-unzip@latest
```

## Usage

```
codepage-unzip -f CHARACTER_ENCODING ZIP_FILENAME
```
CHARACTER_ENCODING is the filename encoding of the zip archive.
This tool internally use iconv for character conversion, so the encoding name should same characterset name of iconv tool.

You may get a list of character encodings with the command `iconv --list`

### Example: decompress a zip with Japanese filenames.

```
codepage-unzip -f SHIFT-JIS japanese_zip_archive.zip
```


