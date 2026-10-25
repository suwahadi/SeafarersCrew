import json
import pathlib

import yaml

here = pathlib.Path(__file__).parent
spec = yaml.safe_load((here / "openapi.yaml").read_text(encoding="utf-8"))
payload = json.dumps(spec, ensure_ascii=False).replace("</", "<\\/")
title = spec["info"]["title"]
page = f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{title}</title>
<style>body{{margin:0}}</style>
</head>
<body>
<div id="docs"></div>
<script src="https://cdn.jsdelivr.net/npm/redoc@2.1.5/bundles/redoc.standalone.js"></script>
<script>
Redoc.init({payload}, {{hideDownloadButton: false, expandResponses: "200,201", pathInMiddlePanel: true}}, document.getElementById("docs"));
</script>
</body>
</html>
"""
(here / "index.html").write_text(page, encoding="utf-8")
