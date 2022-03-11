# Copyright IBM Corporation 2020,2022
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from setuptools import setup

setup(
    name='kar',
    long_description="KAR Python SDK.",
    packages=['kar'],
    install_requires=[
        'hypercorn>=0.13.2', 'fastapi>=0.74.1', 'httpx>=0.22.0',
        'httpx[http2]', 'aiohttp>=3.8.0', 'asyncio>=3.4.3', 'pytest>=6.2.3',
        'pytest-asyncio', 'requests>=2.27.1'
    ],
    version="1.2.1",
    python_requires='>=3.6',
    description='KAR Python SDK.',
    license='Apache 2.0',
    author='KAR authors',
    url='https://github.com/IBM/kar',
    project_urls={
        'Bug Reports': 'https://github.com/IBM/kar/issues',
        'Source': 'https://github.com/IBM/kar',
    },
)
