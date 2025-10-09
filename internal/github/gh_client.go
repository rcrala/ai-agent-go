package githubclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"net/http"
	"strings"
)

type GHClient struct {
	ctx    context.Context
	Token  string
	Repo   string
	Client *http.Client
}

func NewGHClient(ctx context.Context, token, repo string) *GHClient {
	return &GHClient{
		ctx:    ctx,
		Token:  token,
		Repo:   repo,
		Client: &http.Client{},
	}
}

// -----------------------------
// ESTRUCTURAS DE RESPUESTA
// -----------------------------

type GitFile struct {
	SHA string `json:"sha"`
}

// CreateBranch crea una rama temporal desde baseBranch
func (c *GHClient) CreateBranch(tempBranch, baseBranch string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/refs", c.Repo)
	payload := map[string]interface{}{
		"ref": "refs/heads/" + tempBranch,
		"sha": c.getBranchSHA(baseBranch),
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 422 {
		// La rama ya existe, se ignora
		return nil
	}
	if resp.StatusCode >= 300 {
		data, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error creando branch: %s", string(data))
	}
	return nil
}

// -----------------------------
// FUNCIONES AUXILIARES
// -----------------------------
// getBranchSHA obtiene SHA de la rama base
func (c *GHClient) getBranchSHA(branch string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/branches/%s", c.Repo, branch)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.Client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var respJSON map[string]interface{}
	json.Unmarshal(data, &respJSON)
	return respJSON["commit"].(map[string]interface{})["sha"].(string)
}

func (c *GHClient) request(method, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(c.ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(data))
	}

	return io.ReadAll(resp.Body)
}

// -----------------------------
// GESTIÓN DE RAMAS
// -----------------------------

// CreateBranch crea una nueva rama a partir de baseBranch
func (c *GHClient) CreateBranchNew(newBranch, baseBranch string) error {
	// Obtener SHA de la rama base
	urlRef := fmt.Sprintf("https://api.github.com/repos/%s/git/ref/heads/%s", c.Repo, baseBranch)
	body, err := c.request("GET", urlRef, nil)
	if err != nil {
		return fmt.Errorf("error obteniendo ref base: %w", err)
	}

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &refData); err != nil {
		return fmt.Errorf("error parseando JSON de ref base: %w", err)
	}

	// Crear la nueva rama
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/refs", c.Repo)
	payload := map[string]string{
		"ref": "refs/heads/" + newBranch,
		"sha": refData.Object.SHA,
	}
	payloadBytes, _ := json.Marshal(payload)

	_, err = c.request("POST", url, payloadBytes)
	if err != nil {
		if strings.Contains(err.Error(), "Reference already exists") {
			return fmt.Errorf("la rama %s ya existe", newBranch)
		}
		return err
	}

	return nil
}

// -----------------------------
// CREAR / ACTUALIZAR ARCHIVO
// -----------------------------

func (c *GHClient) GetFile(branch, path string) (*GitFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", c.Repo, path, branch)
	data, err := c.request("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var file GitFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return &file, nil
}

// CreateFile crea un archivo nuevo en la rama especificada
func (c *GHClient) CreateFile(branch, path, content string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", c.Repo, path)
	payload := map[string]string{
		"message": "AI Agent: Crear archivo " + path,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  branch,
	}
	payloadBytes, _ := json.Marshal(payload)

	_, err := c.request("PUT", url, payloadBytes)
	return err
}

// UpdateFile actualiza un archivo existente usando su SHA
func (c *GHClient) UpdateFile(branch, path, content, sha string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", c.Repo, path)
	payload := map[string]string{
		"message": "AI Agent: Actualizar archivo " + path,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"sha":     sha,
		"branch":  branch,
	}
	payloadBytes, _ := json.Marshal(payload)

	_, err := c.request("PUT", url, payloadBytes)
	return err
}

// -----------------------------
// PULL REQUEST
// -----------------------------

// CreatePullRequest abre un PR desde sourceBranch hacia baseBranch
func (c *GHClient) CreatePullRequest(sourceBranch, baseBranch, title, body string) (int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls", c.Repo)
	payload := map[string]string{
		"title": title,
		"head":  sourceBranch,
		"base":  baseBranch,
		"body":  body,
	}
	payloadBytes, _ := json.Marshal(payload)

	respData, err := c.request("POST", url, payloadBytes)
	if err != nil {
		return 0, err
	}

	var pr struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(respData, &pr); err != nil {
		return 0, err
	}

	return pr.Number, nil
}

// CreateOrUpdateFileWithPR crea o actualiza un archivo y abre PR
func CreateOrUpdateFileWithPR(ctx context.Context, c *GHClient, tempBranch, baseBranch, filePath, content string) (int, error) {
	// 1️⃣ Crear o actualizar archivo
	existingFile, err := c.GetFile(tempBranch, filePath)
	if err != nil {
		err = c.CreateFile(tempBranch, filePath, content)
	} else {
		err = c.UpdateFile(tempBranch, filePath, content, existingFile.SHA)
	}
	if err != nil {
		return 0, err
	}

	// 2️⃣ Crear Pull Request
	prTitle := "AI Agent: Actualización de arquitectura y cumplimiento"
	prBody := "Se han generado recomendaciones automáticas de arquitectura y código mediante AI Agent y SonarQube."
	prNumber, err := c.CreatePullRequest(tempBranch, baseBranch, prTitle, prBody)
	if err != nil {
		return 0, err
	}
	return prNumber, nil
}