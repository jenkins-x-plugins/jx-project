---
title: API Documentation
linktitle: API Documentation
description: Reference of the jx-promote configuration
weight: 10
---
<p>Packages:</p>
<ul>
<li>
<a href="#project.jenkins-x.io%2fv1alpha1">project.jenkins-x.io/v1alpha1</a>
</li>
</ul>
<h2 id="project.jenkins-x.io/v1alpha1">project.jenkins-x.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#project.jenkins-x.io/v1alpha1.PipelineCatalog">PipelineCatalog</a>
</li><li>
<a href="#project.jenkins-x.io/v1alpha1.Quickstarts">Quickstarts</a>
</li></ul>
<h3 id="project.jenkins-x.io/v1alpha1.PipelineCatalog">PipelineCatalog
</h3>
<p>
<p>PipelineCatalog represents a collection quickstart project</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
project.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>PipelineCatalog</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.PipelineCatalogSpec">
PipelineCatalogSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the PipelineCatalog from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>repositories</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.PipelineCatalogSource">
[]PipelineCatalogSource
</a>
</em>
</td>
<td>
<p>Repositories the repositories containing pipeline catalogs</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="project.jenkins-x.io/v1alpha1.Quickstarts">Quickstarts
</h3>
<p>
<p>Quickstarts represents a collection quickstart project</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
project.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Quickstarts</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartsSpec">
QuickstartsSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the specified quicksatrt configuration</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>quickstarts</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartSource">
[]QuickstartSource
</a>
</em>
</td>
<td>
<p>Quickstarts custom quickstarts to include</p>
</td>
</tr>
<tr>
<td>
<code>defaultOwner</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultOwner the default owner if not specfied</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartImport">
[]QuickstartImport
</a>
</em>
</td>
<td>
<p>Imports import quickstarts from the version stream</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="project.jenkins-x.io/v1alpha1.PipelineCatalogSource">PipelineCatalogSource
</h3>
<p>
(<em>Appears on:</em>
<a href="#project.jenkins-x.io/v1alpha1.PipelineCatalogSpec">PipelineCatalogSpec</a>)
</p>
<p>
<p>PipelineCatalogSource the source of a pipeline catalog</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>label</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gitUrl</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gitRef</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="project.jenkins-x.io/v1alpha1.PipelineCatalogSpec">PipelineCatalogSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#project.jenkins-x.io/v1alpha1.PipelineCatalog">PipelineCatalog</a>)
</p>
<p>
<p>PipelineCatalogSpec defines the desired state of PipelineCatalog.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>repositories</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.PipelineCatalogSource">
[]PipelineCatalogSource
</a>
</em>
</td>
<td>
<p>Repositories the repositories containing pipeline catalogs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="project.jenkins-x.io/v1alpha1.QuickstartImport">QuickstartImport
</h3>
<p>
(<em>Appears on:</em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartsSpec">QuickstartsSpec</a>)
</p>
<p>
<p>QuickstartImport imports quickstats from another folder (such as from the shared version stream)</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>file</code></br>
<em>
string
</em>
</td>
<td>
<p>File file name relative to the root directory to load</p>
</td>
</tr>
<tr>
<td>
<code>includes</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>excludes</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="project.jenkins-x.io/v1alpha1.QuickstartSource">QuickstartSource
</h3>
<p>
(<em>Appears on:</em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartsSpec">QuickstartsSpec</a>)
</p>
<p>
<p>QuickstartSource the source of a quickstart</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Owner</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Version</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Language</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Framework</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Tags</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>DownloadZipURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>GitServer</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>GitKind</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="project.jenkins-x.io/v1alpha1.QuickstartsSpec">QuickstartsSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#project.jenkins-x.io/v1alpha1.Quickstarts">Quickstarts</a>)
</p>
<p>
<p>QuickstartsSpec defines the desired state of Quickstarts.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>quickstarts</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartSource">
[]QuickstartSource
</a>
</em>
</td>
<td>
<p>Quickstarts custom quickstarts to include</p>
</td>
</tr>
<tr>
<td>
<code>defaultOwner</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultOwner the default owner if not specfied</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#project.jenkins-x.io/v1alpha1.QuickstartImport">
[]QuickstartImport
</a>
</em>
</td>
<td>
<p>Imports import quickstarts from the version stream</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>8172249</code>.
</em></p>
