package githubclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	logger "ai-agent-go/internal/logger"
	"net/http"
	"strings"
)

type GHClient struct {
	ctx    context.Context
	Token  string
	Repo   string
	Client *http.Client
	log    *logger.Logger
}

const (
	authTokenPrefix = "token "
	acceptHeader    = "application/vnd.github+json"
)

// when running inside GitHub Actions and the runner cannot perform an operation
// because of workflow permissions, return a helpful hint to troubleshoot.
func actionsPermissionHint() string {
	return "Si esto se ejecuta dentro de GitHub Actions, puede que el token de workflow no tenga permisos para crear o aprobar pull requests.\n" +
		"Opciones de solución:\n" +
		"1) Ajustar permisos del workflow en el archivo de flujo de trabajo (y/o en Settings → Actions → General → Workflow permissions): asegurar 'contents: write' y 'pull-requests: write'.\n" +
		"   Ejemplo YAML:\n" +
		"   permissions:\n" +
		"     contents: write\n" +
		"     pull-requests: write\n" +
		"2) O usar un Personal Access Token (PAT) almacenado en un secret (por ejemplo ACTIONS_PAT) y exportarlo como GITHUB_TOKEN en el job:\n" +
		"   - name: Set PAT as GITHUB_TOKEN\n" +
		"     run: echo \"::add-mask::$ACTIONS_PAT\"; echo \"GITHUB_TOKEN=$ACTIONS_PAT\" >> $GITHUB_ENV\n" +
		"     env:\n" +
		"       ACTIONS_PAT: ${{ secrets.ACTIONS_PAT }}\n" +
		"   Asegúrate de que el PAT tenga scope 'repo' para repos privados y los permisos necesarios.\n" +
		"3) En ejecuciones locales, exportar GITHUB_TOKEN con un PAT antes de ejecutar la herramienta."
}

func NewGHClient(ctx context.Context, token, repo string) *GHClient {
	lg := logger.NewLogger()
	// If no token provided, attempt to fallback to ACTIONS_PAT (useful in GitHub Actions when using a PAT secret)
	if strings.TrimSpace(token) == "" {
		if pat := os.Getenv("ACTIONS_PAT"); pat != "" {
			token = pat
			lg.Info("github", "NewGHClient", "No token supplied, using ACTIONS_PAT from environment")
		}
	}
	lg.Info("github", "NewGHClient", fmt.Sprintf("Inicializando GHClient para repo %s", repo))
	return &GHClient{
		ctx:    ctx,
		Token:  token,
		Repo:   repo,
		Client: &http.Client{},
		log:    lg,
	}
}

// -----------------------------
// ESTRUCTURAS DE RESPUESTA
// -----------------------------

type GitFile struct {
	SHA string `json:"sha"`
}

// CreateBranch is a compatibility wrapper that ensures a branch exists by delegating to CreateBranchNew
func (c *GHClient) CreateBranch(tempBranch, baseBranch string) error {
	return c.CreateBranchNew(tempBranch, baseBranch)
}

// -----------------------------
// FUNCIONES AUXILIARES
// -----------------------------
// Note: getBranchSHA removed — CreateBranchNew uses the refs endpoint and request helper which returns errors

func (c *GHClient) request(method, url string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(c.ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Authorization", authTokenPrefix+c.Token)
	req.Header.Set("Accept", acceptHeader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		if c.log != nil {
			c.log.Error("github", "request", fmt.Sprintf("HTTP request error: %v", err))
		}
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := string(data)
		// if we are running in GitHub Actions, provide an extra hint
		hint := ""
		if os.Getenv("GITHUB_ACTIONS") == "true" || resp.StatusCode == 403 {
			hint = "\n\n" + actionsPermissionHint()
			msg = msg + hint
		}
		if c.log != nil {
			c.log.Error("github", "request", fmt.Sprintf("GitHub API error %d: %s", resp.StatusCode, msg))
		}
		return data, resp.StatusCode, fmt.Errorf("GitHub API error: %s", msg)
	}

	return data, resp.StatusCode, nil
}

// -----------------------------
// GESTIÓN DE RAMAS
// -----------------------------

// CreateBranch crea una nueva rama a partir de baseBranch
func (c *GHClient) CreateBranchNew(newBranch, baseBranch string) error {
	// Obtener SHA de la rama base
	urlRef := fmt.Sprintf("https://api.github.com/repos/%s/git/ref/heads/%s", c.Repo, baseBranch)
	body, status, err := c.request("GET", urlRef, nil)
	if err != nil {
		return fmt.Errorf("error obteniendo ref base (status %d): %w", status, err)
	}

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &refData); err != nil {
		if c.log != nil {
			c.log.Error("github", "CreateBranchNew", fmt.Sprintf("error parseando ref base: %v", err))
		}
		return fmt.Errorf("error parseando JSON de ref base: %w", err)
	}

	// Crear la nueva rama
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/refs", c.Repo)
	payload := map[string]string{
		"ref": "refs/heads/" + newBranch,
		"sha": refData.Object.SHA,
	}
	payloadBytes, _ := json.Marshal(payload)

	// Try to create the ref. If it already exists (HTTP 422), treat as success.
	_, status, err = c.request("POST", url, payloadBytes)
	if err != nil {
		// If the request helper returned a GitHub API error message, inspect it
		if status == 422 || strings.Contains(err.Error(), "Reference already exists") || strings.Contains(err.Error(), "already exists") {
			// branch already exists — not an error for our flow
			if c.log != nil {
				c.log.Info("github", "CreateBranchNew", fmt.Sprintf("La rama %s ya existe, continuando.", newBranch))
			}
			return nil
		}
		if c.log != nil {
			c.log.Error("github", "CreateBranchNew", fmt.Sprintf("error creando branch %s: %v", newBranch, err))
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
	data, status, err := c.request("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed (status %d): %w", status, err)
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

	_, status, err := c.request("PUT", url, payloadBytes)
	if err != nil {
		if c.log != nil {
			c.log.Error("github", "CreateFile", fmt.Sprintf("error creando archivo %s en %s: status %d: %v", path, branch, status, err))
		}
	}
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

	_, status, err := c.request("PUT", url, payloadBytes)
	if err != nil {
		if c.log != nil {
			c.log.Error("github", "UpdateFile", fmt.Sprintf("error actualizando archivo %s en %s: status %d: %v", path, branch, status, err))
		}
	}
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

	respData, status, err := c.request("POST", url, payloadBytes)
	if err != nil {
		if c.log != nil {
			c.log.Error("github", "CreatePullRequest", fmt.Sprintf("error creando PR: status %d: %v", status, err))
		}
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
	// 0️⃣ Asegurar que la rama temporal existe (crear desde baseBranch si no existe)
	if err := c.CreateBranchNew(tempBranch, baseBranch); err != nil {
		// CreateBranchNew already returns a friendly error when branch exists; log and continue
		if c.log != nil {
			c.log.Warning("github", "CreateOrUpdateFileWithPR", fmt.Sprintf("CreateBranchNew result: %v", err))
		}
		// If the error indicates the branch already exists, proceed; otherwise, continue and try operations which will surface the error
	}

	// 1️⃣ Crear o actualizar archivo
	existingFile, err := c.GetFile(tempBranch, filePath)
	if err != nil {
		// If GetFile returned 404 because branch didn't exist or file not found, try to create the file
		if c.log != nil {
			c.log.Info("github", "CreateOrUpdateFileWithPR", fmt.Sprintf("File not found on branch %s: attempting to create it", tempBranch))
		}
		err = c.CreateFile(tempBranch, filePath, content)
	} else {
		err = c.UpdateFile(tempBranch, filePath, content, existingFile.SHA)
	}
	if err != nil {
		if c.log != nil {
			c.log.Error("github", "CreateOrUpdateFileWithPR", fmt.Sprintf("error creando/actualizando archivo: %v", err))
		}
		return 0, err
	}

	// 2️⃣ Crear Pull Request
	prTitle := "AI Agent: Actualización de arquitectura y cumplimiento"
	prBody := "Se han generado recomendaciones automáticas de arquitectura y código mediante AI Agent y SonarQube."
	prNumber, err := c.CreatePullRequest(tempBranch, baseBranch, prTitle, prBody)
	if err != nil {
		if c.log != nil {
			c.log.Error("github", "CreateOrUpdateFileWithPR", fmt.Sprintf("error creando PR: %v", err))
		}
		return 0, err
	}
	if c.log != nil {
		c.log.Info("github", "CreateOrUpdateFileWithPR", fmt.Sprintf("PR creado: %d", prNumber))
	}
	return prNumber, nil
}
