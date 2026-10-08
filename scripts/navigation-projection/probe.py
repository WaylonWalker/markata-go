"""Offline canonical-region byte probe, never a production publisher.

Lifecycle, outside-wrapper widgets and inline setup are incomplete. The footer
contains route state and is excluded from the experimental persistent shell.
The shell hash does not yet cover the production runtime lifecycle contract.
"""
from html.parser import HTMLParser
import argparse
import gzip
import hashlib
import html
import json
import pathlib
from urllib.parse import urlsplit

class CanonicalRanges(HTMLParser):

    def __init__(self, s):
        super().__init__(convert_charrefs=False)
        self.s = s
        self.off = [0]
        self.depth = 0
        self.start = None
        self.end = None
        self.scripts = []
        self.assets = []
        self.head = False
        self.header = False
        self.headerstart = 0
        self.headerend = 0
        for line in s.splitlines(keepends=True):
            self.off.append(self.off[-1] + len(line))

    def pos(self):
        l, c = self.getpos()
        return self.off[l - 1] + c

    def handle_starttag(self, t, a):
        d = dict(a)
        if t == 'div':
            if d.get('id') == 'view-transition-page':
                self.start = self.pos()
                self.depth = 1
            elif self.depth:
                self.depth += 1
        if t == 'html':
            self.htmltag = self.get_starttag_text()
            self.htmlattrs = d
        if t == 'head':
            self.headstart = self.pos()
        if t == 'title':
            self.titlestart = self.pos() + len(self.get_starttag_text())
        if t == 'link' and d.get('rel') == 'canonical':
            self.canonical = d.get('href')
        if t == 'body':
            self.bodyattrs = d
        if t == 'footer' and 'site-footer' in d.get('class', '').split():
            self.footerstart = self.pos()
        if t == 'body':
            self.body = self.get_starttag_text()
        if t == 'header' and self.headerstart == 0:
            self.headerstart = self.pos()
        if t == 'script':
            self.scripts.append(d)
        if t == 'link' and d.get('rel') == 'stylesheet':
            self.assets.append(d)

    def handle_endtag(self, t):
        if t == 'div' and self.depth:
            self.depth -= 1
            if not self.depth:
                self.end = self.pos() + len('</div>')
        if t == 'head':
            self.headend = self.pos() + 7
        if t == 'header' and (not self.headerend):
            self.headerend = self.pos() + 9
        if t == 'title':
            self.title = html.unescape(self.s[self.titlestart:self.pos()])
        if t == 'footer' and hasattr(self, 'footerstart') and (not hasattr(self, 'footerend')):
            self.footerend = self.pos() + 9

def digest(data):
    return hashlib.sha256(data.encode()).hexdigest()

def projection(source, root):
    p = CanonicalRanges(source)
    p.feed(source)
    if p.start is None or p.end is None:
        raise ValueError('missing canonical route wrapper')
    region = source[p.start:p.end]
    head = source[p.headstart:p.headend]
    required = []
    for descriptor in p.assets + p.scripts:
        item = dict(descriptor)
        url = item.get('href') or item.get('src')
        if url and url.startswith('/'):
            candidate = (root / urlsplit(url).path.lstrip('/')).resolve()
            if not candidate.is_relative_to(root):
                raise ValueError('asset escapes source output')
            if candidate.is_file():
                item['content_sha256'] = hashlib.sha256(candidate.read_bytes()).hexdigest()
        required.append(item)
    shell = digest('nav-probe-v1\x00' + source[p.headerstart:p.headerend])
    meta = {'schema': 'nav-v1', 'shell': shell, 'title': p.title, 'canonical': getattr(p, 'canonical', None), 'html_attributes': p.htmlattrs, 'body_attributes': p.bodyattrs, 'required_assets': required}
    serialized = json.dumps(meta, sort_keys=True, separators=(',', ':')).replace('<', '\\u003c')
    revision = digest('nav-v1\x00' + region + '\x00' + head + '\x00' + serialized)
    opening = p.htmltag[:-1] + ' data-nav-schema="nav-v1" data-nav-shell="' + shell + '" data-nav-revision="' + revision + '">'
    artifact = '<!doctype html>' + opening + head + p.body + region + '<script type="application/json" id="nav-probe-metadata">' + serialized + '</script></body></html>'
    check = CanonicalRanges(artifact)
    check.feed(artifact)
    assert artifact[check.start:check.end] == region
    assert check.title == p.title and check.bodyattrs == p.bodyattrs
    return (artifact, revision, shell, len(region.encode()))

def main():
    parser = argparse.ArgumentParser(description='Generate offline canonical-region byte probes; not a production publisher.')
    parser.add_argument('--version', action='version', version='navigation-byte-probe 1')
    parser.add_argument('source', type=pathlib.Path, help='Existing generated site output (read only)')
    parser.add_argument('fixtures', type=pathlib.Path, help='Separate destination for transport fixtures')
    parser.add_argument('--routes', default='vimgrep-open-buffers,til,2025-nas,tmux-pop-size,bloatware-is-dying')
    args = parser.parse_args()
    root = args.source.resolve()
    out = args.fixtures.resolve()
    if out == root or out.is_relative_to(root):
        parser.error('fixtures must be outside source output')
    rows = []
    for route in args.routes.split(','):
        path = (root / route / 'index.html').resolve()
        if not path.is_relative_to(root):
            parser.error('route escapes source output')
        source = path.read_text()
        artifact, revision, shell, region_size = projection(source, root)
        destination = (out / route).resolve()
        if not destination.is_relative_to(out):
            parser.error('route escapes fixtures')
        destination.mkdir(parents=True, exist_ok=True)
        (destination / 'index.html').write_text(source)
        (destination / '_nav.html').write_text(artifact)
        before = len(gzip.compress(source.encode()))
        after = len(gzip.compress(artifact.encode()))
        rows.append({'route': route, 'full_bytes': len(source.encode()), 'projection_bytes': len(artifact.encode()), 'region_bytes': region_size, 'full_gzip': before, 'projection_gzip': after, 'saving_gzip': before - after, 'revision': revision, 'shell': shell})
    print(json.dumps({'method': 'offline broad canonical region + canonical head + metadata; lifecycle incomplete; incomplete envelope', 'routes': rows}, indent=2))
if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, AssertionError) as error:
        import sys
        print('navigation-byte-probe: ' + str(error), file=sys.stderr)
        sys.exit(1)
