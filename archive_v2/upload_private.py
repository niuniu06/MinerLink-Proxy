import requests, json, sys

headers={'Authorization': 'token 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', 'Accept': 'application/vnd.github.v3+json'}
repo='niuniu06/MinerLink-Proxy'
tag='v2.0.72-beta'

r = requests.get(f'https://api.github.com/repos/{repo}/releases/tags/{tag}', headers=headers)
if r.status_code == 200:
    release_id = r.json().get('id')
    requests.delete(f'https://api.github.com/repos/{repo}/releases/{release_id}', headers=headers)

res = requests.post(f'https://api.github.com/repos/{repo}/releases', headers=headers, json={'tag_name': tag, 'name': tag, 'body': 'MinerLink-Proxy Private Release v2.0.66-beta', 'draft': False, 'prerelease': False}).json()

upload_url = res.get('upload_url', '').replace('{?name,label}', '')

requests.post(upload_url+'?name=MinerLink-Proxy-Linux.zip', headers={'Authorization': headers['Authorization'], 'Content-Type': 'application/zip'}, data=open('MinerLink-Proxy-Linux.zip','rb').read())
requests.post(upload_url+'?name=MinerLink-Proxy-Windows.zip', headers={'Authorization': headers['Authorization'], 'Content-Type': 'application/zip'}, data=open('MinerLink-Proxy-Windows.zip','rb').read())
requests.post(upload_url+'?name=install.sh', headers={'Authorization': headers['Authorization'], 'Content-Type': 'application/x-sh'}, data=open('../MinerLink-Public/install.sh','rb').read())

print('Done!')
