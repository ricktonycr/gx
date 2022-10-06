package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/gorilla/mux"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	parser "github.com/ricktonycr/gx_parser"
)

type Usuarios struct {
	Usuarios []Book `xml:"CodeBlock"`
}

const path = "/usr/local/bin/wkhtmltopdf"

type Book struct {
	LeftT   string `xml:"left"`
	ShapeT  string `xml:"Shape"`
	SourceT []byte `xml:"Source"`
}

type TreeShapeListener struct {
	*parser.BaseGXParserListener
}

type Code struct {
	Text string
}

type Data struct {
	FST900     []FST900
	FST200     []FST200
	FST098     []FST098
	PSEUDOCODE string
}

type FST900 struct {
	PGMNOM string
	DESC   string
}

type FST200 struct {
	OPGCOD string
	DESC   string
}

type FST098 struct {
	TPCOD  string
	TPCORR string
	DESC   string
}

var forBlocks int
var forTable string
var variablesC int
var variables [9999]string
var values [9999]string

// Tablas
var fst900n int = -1
var fst900s []FST900
var fst200n int = -1
var fst200s []FST200
var fst098n int = -1
var fst098s []FST098
var pseudocode string

func writeLineln(text string) {
	for i := 0; i < forBlocks; i++ {
		//fmt.Print("  ")
		pseudocode += "  "
	}
	//fmt.Println(text)
	pseudocode += text + "\n"
}

func checkTable() {
	forTable = strings.Trim(forTable, "-")
	forTable = strings.Trim(forTable, " ")
	forTable = strings.Trim(forTable, "\n")
	switch forTable {
	case "FST900":
		fst900n++
		fst900s = append(fst900s, FST900{"", ""})
	case "FST200":
		fst200n++
		fst200s = append(fst200s, FST200{"", ""})
	case "FST098":
		fst098n++
		fst098s = append(fst098s, FST098{"", "", ""})
	}
}

func getVar(str string) string {
	f := strings.HasPrefix(str, "&")
	str = strings.ToUpper(str)
	val := ""
	if f {
		for i, value := range variables {
			if value == str {
				val = values[i]
			}
			if i == variablesC {
				break
			}
		}
	} else {
		val = str
	}
	return val
}

func checkOrigen(atrib string, origen string) {
	origen = getVar(origen)
	switch forTable {
	case "FST900":
		pgmnomTemp := ""
		descTemp := ""
		switch atrib {
		case "PGMNOM":
			pgmnomTemp = origen
		}
		if pgmnomTemp != "" {
			fst900s[fst900n].PGMNOM = pgmnomTemp
		}
		if descTemp != "" {
			fst900s[fst900n].DESC = descTemp
		}
	case "FST200":
		opgcodTemp := ""
		descTemp := ""
		switch atrib {
		case "OPGCOD":
			opgcodTemp = origen
		}
		if opgcodTemp != "" {
			fst200s[fst200n].OPGCOD = opgcodTemp
		}
		if descTemp != "" {
			fst200s[fst200n].DESC = descTemp
		}
	case "FST098":
		tpcodTemp := ""
		tpcorrTemp := ""
		descTemp := ""
		switch atrib {
		case "TPCOD":
			tpcodTemp = origen
		case "TPCORR":
			tpcorrTemp = origen
		}
		if tpcodTemp != "" {
			fst098s[fst098n].TPCOD = tpcodTemp
		}
		if tpcorrTemp != "" {
			fst098s[fst098n].TPCORR = tpcorrTemp
		}
		if descTemp != "" {
			fst098s[fst098n].DESC = descTemp
		}
	}

}

func getFullText(ctx antlr.ParserRuleContext) string {
	if ctx.GetStart() == nil || ctx.GetStop() == nil || ctx.GetStart().GetStart() < 0 || ctx.GetStop().GetStop() < 0 {
		return ctx.GetText()
	}
	return ctx.GetStart().GetInputStream().GetText(ctx.GetStart().GetStart(), ctx.GetStop().GetStop())
}

func NewTreeShapeListener() *TreeShapeListener {
	return new(TreeShapeListener)
}

// EnterDocLine is called when production docLine is entered.
func (s *TreeShapeListener) EnterDocLine(ctx *parser.DocLineContext) {
	if ctx.GetInfo() != nil {
		txt := ctx.GetInfo().GetText()
		txt = strings.Trim(strings.TrimLeft(txt[3:], " "), "\n")
		writeLineln(txt)
	}
	if ctx.GetTag() != nil {
		txt := ctx.GetTag().GetText()
		txt = strings.TrimLeft(txt[3:], " ")
		space := strings.Index(txt, " ")
		//table := txt[0:space]
		rest := strings.Trim(strings.Trim(txt[space:], " "), "\n")
		writeLineln(rest + "TAG")
	}
}

// ExitDocLine is called when production docLine is exited.
func (s *TreeShapeListener) ExitDocLine(ctx *parser.DocLineContext) {}

func (tree *TreeShapeListener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	//fmt.Print("EnterEveryRule: ")
	//fmt.Println(ctx.GetText())
}

func (s *TreeShapeListener) EnterStatement(ctx *parser.StatementContext) {
	//fmt.Print("EnterStatement: ")
	//fmt.Println(ctx.GetText())
	value := ""
	if ctx.GetVariable() != nil {
		if ctx.GetExpresion() != nil {
			expresion := ctx.GetExpresion()
			if expresion.GetCadena() != nil {
				value = expresion.GetCadena().GetText()
			} else if expresion.GetDecimal() != nil {
				value = expresion.GetDecimal().GetText()
			}
		}
		if value != "" {
			variables[variablesC] = strings.ToUpper(ctx.GetVariable().GetText())
			values[variablesC] = value
			variablesC = variablesC + 1
		}
	}
}

// EnterGxcode is called when production gxcode is entered.
func (s *TreeShapeListener) EnterGxcode(ctx *parser.GxcodeContext) {

}

// ExitGxcode is called when production gxcode is exited.
func (s *TreeShapeListener) ExitGxcode(ctx *parser.GxcodeContext) {

}

func (s *TreeShapeListener) ExitStatement(ctx *parser.StatementContext) {
	//fmt.Print("ExitStatement: ")
	// fmt.Println(ctx.GetText())
}

func (s *TreeShapeListener) EnterForBlock(ctx *parser.ForBlockContext) {
	//fmt.Print("EnterForBlock: ")
	// fmt.Println(ctx.GetText())
	tabla := "####"
	indices := ctx.GetIndices()
	condiciones := ctx.GetCondiciones()

	if ctx.GetDoc() != nil {
		doc := ctx.GetDoc().GetText()
		docq := strings.Trim(doc[3:], " ")
		docq = strings.Trim(docq, "\n")
		writeLineln(docq)
	} else {

		// Obtengo el nombre de la tabla
		if ctx.GetComentario() != nil {
			comentario := ctx.GetComentario().GetText()
			comentarioq := strings.TrimLeft(comentario[2:], " ")
			palabras := strings.Split(comentarioq, ":")

			if len(palabras) > 0 {
				tabla = palabras[0]
			} else {
				palabras = strings.Split(comentarioq, "\n")
				if len(palabras) > 0 {
					tabla = palabras[0]
				}
			}
		}
		tabla = strings.ToUpper(tabla)
		forTable = tabla
		checkTable()

		// En caso de FOR TO
		if ctx.GetDesde() != nil {
			linea := "Inicio Recorrido Desde " + strings.Trim(ctx.GetDesde().GetText(), " ") + " Hasta " + strings.Trim(ctx.GetHasta().GetText(), " ")
			if ctx.GetCada() != nil {
				linea = linea + " Cada " + strings.Trim(ctx.GetCada().GetText(), " ")
			}
			linea = linea + " EN " + strings.Trim(ctx.GetEn().GetText(), " ")
			writeLineln(linea)
		} else {
			if ctx.GetSdt() != nil {
				linea := "Inicio Recorrido SDT " + strings.Trim(ctx.GetSdt().GetText(), " ")
				writeLineln(linea)
			} else {
				writeLineln("Inicio Recorrido Tabla " + tabla)
				if len(indices) > 0 {
					temp := ("Índice ")
					for i := 0; i < len(indices); i++ {
						if i == 0 {
							temp += strings.ToUpper(indices[i].GetText())
						} else {
							temp += "," + strings.ToUpper(indices[i].GetText())
						}
					}
					writeLineln(temp)
				}

				if len(condiciones) > 0 {
					temp := "Filtra Por "
					for i := 0; i < len(condiciones); i++ {
						where := condiciones[i]
						cond := where.GetCondicion()
						expresiones := cond.GetExpresions()
						atributo := ""
						origen := ""
						if len(expresiones) > 0 {
							if expresiones[0].GetAtributo() != nil {
								atributo = strings.ToUpper(expresiones[0].GetAtributo().GetText())

								if expresiones[1].GetVariable() != nil {
									origen = expresiones[1].GetVariable().GetText()
									checkOrigen(atributo, origen)
								} else if expresiones[1].GetCadena() != nil {
									origen = expresiones[1].GetCadena().GetText()
									checkOrigen(atributo, origen)
								}
							}

							if i == 0 {
								temp += atributo
							} else {
								temp += "," + atributo
							}
						}

					}
					if temp != "Filtra Por " {
						writeLineln(temp)
					}
				}
			}
		}
	}

	forBlocks++
}

func (s *TreeShapeListener) ExitForBlock(ctx *parser.ForBlockContext) {
	//fmt.Print("ExitForBlock: ")
	forBlocks--
	if ctx.GetDoc() == nil {
		writeLineln("Fin Recorrido")
	}
}

func (s *TreeShapeListener) EnterFuncion(ctx *parser.FuncionContext) {
	//fmt.Print("EnterFuncion: ")
	// fmt.Println(ctx.GetText())
}

func (s *TreeShapeListener) ExitFuncion(ctx *parser.FuncionContext) {
	//fmt.Print("ExitFuncion: ")
	// fmt.Println(ctx.GetText())
}

func (s *TreeShapeListener) EnterSubrutine(ctx *parser.SubrutineContext) {
	linea := "SUBRUTINA "
	linea = linea + ctx.GetNombre().GetText()
	writeLineln(linea)
	forBlocks++
}

// ExitSubrutine is called when production subrutine is exited.
func (s *TreeShapeListener) ExitSubrutine(ctx *parser.SubrutineContext) {
	forBlocks--
}

// EnterIfBlock is called when production ifBlock is entered.
func (s *TreeShapeListener) EnterIfBlock(ctx *parser.IfBlockContext) {
	//fmt.Print("EnterIfBlock: ")
	// fmt.Println(ctx.GetText())
	linea := "SI "
	if ctx.GetComentario() != nil {
		comentario := ctx.GetComentario().GetText()
		comentario = comentario[3:]
		linea = linea + strings.Trim(comentario, " ")
	} else {
		linea = linea + getFullText(ctx.GetCondicion())
	}

	writeLineln(linea)
	forBlocks++
}

// ExitIfBlock is called when production ifBlock is exited.
func (s *TreeShapeListener) ExitIfBlock(ctx *parser.IfBlockContext) {
	//fmt.Print("ExitIfBlock: ")
	// fmt.Println(ctx.GetText())
	forBlocks--
	linea := "Fin Si"
	writeLineln(linea)
}

func (s *TreeShapeListener) EnterNewBlock(ctx *parser.NewBlockContext) {
	//fmt.Print("EnterNewBlock: ")
	// fmt.Println(ctx.GetText())
	tabla := "####"

	// Obtengo el nombre de la tabla
	if ctx.GetComentario() != nil {
		comentario := ctx.GetComentario().GetText()
		comentarioq := strings.TrimLeft(comentario[2:], " ")
		palabras := strings.Split(comentarioq, ":")
		if len(palabras) > 0 {
			tabla = palabras[0]
		}
	}
	tabla = strings.ToUpper(tabla)
	linea := "Nuenvo Registro en la Tabla " + tabla
	writeLineln(linea)
	forBlocks++
}

func (s *TreeShapeListener) ExitNewBlock(ctx *parser.NewBlockContext) {
	//fmt.Print("ExitNewBlock: ")
	// fmt.Println(ctx.GetText())
	forBlocks--
}

func (tree *TreeShapeListener) VisitTerminal(node antlr.TerminalNode) {
	word := strings.ToUpper(node.GetText())
	switch word {
	case "Else":
		forBlocks--
		writeLineln("Si No")
		forBlocks++
	}
	//fmt.Print("Nodo: ")
	//fmt.Println(word)
}

// EnterDoWhile is called when production doWhile is entered.
func (s *TreeShapeListener) EnterDoWhile(ctx *parser.DoWhileContext) {
	linea := "Recorrido Mientras "
	if ctx.GetComentario() != nil {
		comentario := ctx.GetComentario().GetText()
		comentario = comentario[3:]
		linea = linea + strings.Trim(comentario, " ")
	} else {
		linea = linea + getFullText(ctx.GetCondicion())
	}

	writeLineln(linea)
	forBlocks++
}

// ExitDoWhile is called when production doWhile is exited.
func (s *TreeShapeListener) ExitDoWhile(ctx *parser.DoWhileContext) {
	forBlocks--
	linea := "Fin Recorrido"
	writeLineln(linea)
}

func main1() {
	input, _ := antlr.NewFileStream(os.Args[1])
	lexer := parser.NewGXLexer(input)

	/*
		array := lexer.GetAllTokens()
		for i := 0; i < len(array); i++ {
			fmt.Println(array[i].GetText())
		}
	*/
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewGXParser(stream)
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true
	tree := p.Gxcode()
	antlr.ParseTreeWalkerDefault.Walk(NewTreeShapeListener(), tree)

	writeLineln("fst900: " + strconv.Itoa(fst900n))

	for i, val := range fst900s {
		if fst900n == -1 {
			break
		}
		fmt.Print("FST900: ")
		fmt.Println(val)
		if i == fst900n {
			break
		}
	}

	for i, val := range fst200s {
		if fst200n == -1 {
			break
		}
		fmt.Print("FST200: ")
		fmt.Println(val)
		if i == fst200n {
			break
		}
	}

	for i, val := range fst098s {
		if fst098n == -1 {
			break
		}
		fmt.Print("FST098: ")
		fmt.Println(val)
		if i == fst098n {
			break
		}
	}

	//fmt.Println(pseudocode)

	response := Data{fst900s, fst200s, fst098s, pseudocode}

	fmt.Println(response)

}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
}

func code(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var code Code
	fmt.Println(reqBody)
	json.Unmarshal(reqBody, &code)
	fmt.Println(code)
	text := code.Text
	fmt.Println(text)

	if text != "" {

		input := antlr.NewInputStream(text)
		lexer := parser.NewGXLexer(input)

		stream := antlr.NewCommonTokenStream(lexer, 0)
		p := parser.NewGXParser(stream)
		p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
		p.BuildParseTrees = true
		tree := p.Gxcode()
		antlr.ParseTreeWalkerDefault.Walk(NewTreeShapeListener(), tree)

		//fmt.Println(pseudocode)

		response := Data{fst900s, fst200s, fst098s, pseudocode}

		//fmt.Println(response)
		json.NewEncoder(w).Encode(response)
	}
}

func upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("File Upload Endpoint Hit")
	objetivo := r.FormValue("objetivo")
	funcional := r.FormValue("funcional")
	fst900s = nil
	fst200s = nil
	fst098s = nil
	fst900n = -1
	fst200n = -1
	fst098n = -1
	pseudocode = ""
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("temp-files", "upload-*")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	// Descomprimir el contenido
	nameXPZ := tempFile.Name()
	zipR, err := zip.OpenReader(nameXPZ)
	if err != nil {
		panic(err)
	}
	codigo := ""
	for _, file := range zipR.File {
		filePath := file.Name
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			panic(err)
		}
		fileInArchive, err := file.Open()
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()

		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		decodingReader := transform.NewReader(file, charmap.Windows1252.NewDecoder())

		lines := ""

		scanner := bufio.NewScanner(decodingReader)
		for scanner.Scan() {
			lines += scanner.Text() + "\n"
		}

		/*
			fmt.Println("Leyendo: " + file.Name)
			r, err := file.Open()
			if err != nil {
				log.Fatal(err)
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			newStr := buf.String()*/
		//newStr, err = charmap.Windows1252.NewDecoder().String(newStr)

		newStr := lines
		if err != nil {
			panic(err)
		}
		length := len(newStr)
		fmt.Println(length)

		// Buscando <Codeblock> y </Codeblock>
		codeblockStr := "<CodeBlock>"
		codeblockStr2 := "</CodeBlock>"
		recortarI := 0
		recortarF := 0
		strTemp := ""
		txtSource := ""
		codigo = ""
		i := 0
		for {
			// fmt.Println(i)
			if string(newStr[i]) == "<" {
				if i+len(codeblockStr) <= len(newStr) {
					strTemp = newStr[i : i+len(codeblockStr)]
				}
				// fmt.Printf(strTemp)
				if strTemp == codeblockStr {
					recortarI = i
				}
			}
			if recortarI != 0 {
				if string(newStr[i]) == "<" {
					strTemp = newStr[i : i+len(codeblockStr2)]
					// fmt.Println("strTemp: ", strTemp)
					if strTemp == codeblockStr2 {
						recortarF = i
						// fmt.Println("recortarI: ", recortarI, "recortarF: ", recortarF)
					}
				}
			}
			if recortarI != 0 && recortarF != 0 {
				// Buscar <Source> y </Source>
				txtSource = newStr[recortarI:recortarF+len(codeblockStr2)] + "\n"
				codigo = codigo + "\n" + buscarCod(txtSource)
				recortarI = 0
				recortarF = 0
			}
			i = i + 1
			if i == length {
				break
			}
		}
		//fmt.Println(codigo)
		// Grabando el string en txt
		rec := []byte(codigo)
		name := "codigo.txt" // "source.xml"

		os.Remove(name)

		err = ioutil.WriteFile(name, rec, 0644)
		if err != nil {
			log.Fatal(err)
		}
		//err = r.Close()
		if err != nil {
			panic(err)
		}
	}
	//codigo = fromWindows1252(codigo)
	input := antlr.NewInputStream(codigo)
	//fmt.Println(input)

	if true {

		lexer := parser.NewGXLexer(input)
		stream := antlr.NewCommonTokenStream(lexer, 0)
		p := parser.NewGXParser(stream)
		p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
		p.BuildParseTrees = true
		tree := p.Gxcode()
		antlr.ParseTreeWalkerDefault.Walk(NewTreeShapeListener(), tree)

		//fmt.Println(pseudocode)

		response := Data{fst900s, fst200s, fst098s, pseudocode}

		//fmt.Println(response)
		//file, _ := json.MarshalIndent(response, "", " ")

		fmt.Println(response)
		os.Remove("test.json")
		//_ = ioutil.WriteFile("test.json", file, 0644)

		topFile, err := os.Open("html/top.html")
		if err != nil {
			panic(err)
		}
		bottomFile, err := os.Open("html/bottom.html")
		if err != nil {
			panic(err)
		}

		top, err := ioutil.ReadAll(topFile)
		topS := string(top)
		bot, err := ioutil.ReadAll(bottomFile)
		botS := string(bot)

		all := topS
		// Titulo objetivos
		all = all + "<div style='mso-element:para-border-div;border:none;border-bottom:solid windowtext 1.5pt;padding:0cm 0cm 1.0pt 0cm'><h1><a name=\"_Toc40123012\"><span lang=ES>Objetivo</span></a></h1></div>"
		// Contenido objetivos
		all = all + "<p class=MsoNormal style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:0cm;text-align:justify;text-justify:inter-ideograph'><span lang=ES-TRAD style='color:#241F0C;mso-ansi-language:ES-TRAD'>" + objetivo + "<o:p></o:p></span></p>"
		all = all + "<br><br>"
		// Titulo pre requisitos
		all = all + "<div style='mso-element:para-border-div;border:none;border-bottom:solid windowtext 1.5pt;padding:0cm 0cm 1.0pt 0cm'><h1><a name=\"_Toc40123013\"><span lang=ES>Pre - requisitos</span></a><span lang=ES> <span style='mso-tab-count:1'> </span></span></h1></div>"
		all = all + "<br><br><br>"
		// Titulo instrucciones de instalación
		all = all + "<div style='mso-element:para-border-div;border:none;border-bottom:solid windowtext 1.5pt;padding:0cm 0cm 1.0pt 0cm'><h1><a name=\"_Toc40123014\"><span lang=ES>Instrucciones de instalación</span></a><span lang=ES><span style='mso-tab-count:1'> </span></span></h1></div>"

		pasos := 1
		figuras := 1
		fst200T := response.FST200
		fst900T := response.FST900

		for _, item := range fst200T {
			all = all + "<h3><a name=\"_Toc40123015\"><span lang=ES>Paso " + strconv.Itoa(pasos) + " Opción General " + strings.Trim(item.OPGCOD, " ") + "</span></a></h3>"
			texto := "Para habilitar el funcionamiento ... , es necesario habilitar la opción general " + strings.Trim(item.OPGCOD, " ") + " con la siguiente información:"
			all = all + "<p class=MsoNormal><span lang=ES style='mso-fareast-language:ES'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br>"
			texto = "Empresa: 1"
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			texto = "Código de opción general: " + strings.Trim(item.OPGCOD, " ")
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			texto = "Descripción: " + strings.Trim(item.DESC, " ")
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br>"
			texto = "A continuación, se muestra un ejemplo de la habilitación de una opción general. Se ingresa al panel de Mantenimiento de Opciones generales (panel HTRT200) y se presiona el botón \"Agregar\"."
			all = all + "<p class=MsoNormal><span lang=ES style='mso-fareast-language:ES'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br><br>"
			all = all + "<p class=Figura1 style='text-indent:-35.7pt;mso-text-indent-alt:-17.85pt;mso-list:l40 level1 lfo47'><![if !supportLists]><span lang=ES style='mso-fareast-font-family:\"Franklin Gothic Book\";mso-bidi-font-family:\"Franklin Gothic Book\"'><span style='mso-list:Ignore'><span style='font:7.0pt \"Times New Roman\"'></span>Figura " + strconv.Itoa(figuras) + " - Mantenimiento de Opciones Generales (panel HTRT200)</span></span><![endif]><span lang=ES><o:p>&nbsp;</o:p></span></p>"
			figuras++
			all = all + "<br><br>"
			texto = "Luego se ingresa la información solicitada por el panel según se muestra en la imagen y se presiona el botón \"Confirmar\"."
			all = all + "<p class=MsoNormal><span lang=ES style='mso-fareast-language:ES'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br><br>"
			all = all + "<p class=Figura1 style='text-indent:-35.7pt;mso-text-indent-alt:-17.85pt;mso-list:l40 level1 lfo47'><![if !supportLists]><span lang=ES style='mso-fareast-font-family:\"Franklin Gothic Book\";mso-bidi-font-family:\"Franklin Gothic Book\"'><span style='mso-list:Ignore'><span style='font:7.0pt \"Times New Roman\"'></span>Figura " + strconv.Itoa(figuras) + " - Agregar Opción General (panel HTRT200T)</span></span><![endif]><span lang=ES><o:p>&nbsp;</o:p></span></p>"
			figuras++
			all = all + "<br><br>"
			pasos++
		}

		for _, item := range fst900T {
			all = all + "<h3><a name=\"_Toc40123015\"><span lang=ES>Paso " + strconv.Itoa(pasos) + " Programa particular " + strings.Trim(item.PGMNOM, " ") + "</span></a></h3>"
			texto := "Para habilitar el funcionamiento ... , es necesario habilitar el programa particular " + strings.Trim(item.PGMNOM, " ") + " con la siguiente información:"
			all = all + "<p class=MsoNormal><span lang=ES style='mso-fareast-language:ES'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br>"
			texto = "Empresa: 1"
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			texto = "Programa particular: " + strings.Trim(item.PGMNOM, " ")
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			texto = "Programa llamado: " + strings.Trim(item.PGMNOM, " ")
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			texto = "Descripción: " + strings.Trim(item.DESC, " ")
			all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br>"
			texto = "A continuación, se muestra un ejemplo de la habilitación de un programa particular. Se ingresa al panel de Mantenimiento de Programas particulares (panel HTRT900) y se presiona el botón \"Agregar\"."
			all = all + "<p class=MsoNormal><span lang=ES style='mso-fareast-language:ES'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br><br>"
			all = all + "<p class=Figura1 style='text-indent:-35.7pt;mso-text-indent-alt:-17.85pt;mso-list:l40 level1 lfo47'><![if !supportLists]><span lang=ES style='mso-fareast-font-family:\"Franklin Gothic Book\";mso-bidi-font-family:\"Franklin Gothic Book\"'><span style='mso-list:Ignore'><span style='font:7.0pt \"Times New Roman\"'></span>Figura " + strconv.Itoa(figuras) + " - Mantenimiento de Programas Particulares (panel HTRT900)</span></span><![endif]><span lang=ES><o:p>&nbsp;</o:p></span></p>"
			figuras++
			all = all + "<br><br>"
			texto = "Luego se ingresa la información solicitada por el panel según se muestra en la imagen y se presiona el botón \"Confirmar\"."
			all = all + "<p class=MsoNormal><span lang=ES style='mso-fareast-language:ES'>" + texto + "<o:p></o:p></span></p>"
			all = all + "<br><br>"
			all = all + "<p class=Figura1 style='text-indent:-35.7pt;mso-text-indent-alt:-17.85pt;mso-list:l40 level1 lfo47'><![if !supportLists]><span lang=ES style='mso-fareast-font-family:\"Franklin Gothic Book\";mso-bidi-font-family:\"Franklin Gothic Book\"'><span style='mso-list:Ignore'><span style='font:7.0pt \"Times New Roman\"'></span>Figura " + strconv.Itoa(figuras) + " - Agregar Programa Particular (panel HTRT900T)</span></span><![endif]><span lang=ES><o:p>&nbsp;</o:p></span></p>"
			figuras++
			all = all + "<br><br>"
			pasos++
		}

		// Titulo instrucciones de uso
		all = all + "<div style='mso-element:para-border-div;border:none;border-bottom:solid windowtext 1.5pt;padding:0cm 0cm 1.0pt 0cm'><h1><a name=\"_Toc40123017\"><span lang=ES>Instrucciones de uso</span></a></h1></div>"
		all = all + "<br><br>"
		// Titulo especificación funcional
		all = all + "<div style='mso-element:para-border-div;border:none;border-bottom:solid windowtext 1.5pt;padding:0cm 0cm 1.0pt 0cm'><h1><a name=\"_Toc40123018\"><span lang=ES>Especificación funcional</span></a></h1></div>"
		all = all + "<p class=MsoNormal style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:0cm;text-align:justify;text-justify:inter-ideograph'><span lang=ES-TRAD style='color:#241F0C;mso-ansi-language:ES-TRAD'>" + funcional + "<o:p></o:p></span></p>"
		all = all + "<br><br>"
		// Titulo especificación técnica
		all = all + "<div style='mso-element:para-border-div;border:none;border-bottom:solid windowtext 1.5pt;padding:0cm 0cm 1.0pt 0cm'><h1><a name=\"_Toc40123019\"><span lang=ES>Especificación técnica</span></a></h1></div>"
		all = all + botS
		all = all + "<br><br>"
		// Pseudocodigo
		Nombre := "PRC01TEST - Proceso de Cancelación"
		all = all + "<h3><a name=\"_Toc40123020\"><span lang=ES>Pseudocódigo</span></a></h3><p class=MsoNormal style='text-align:justify;text-justify:inter-ideograph;tab-stops:121.5pt'><b><span lang=ES style='color:#241F0C'>" + Nombre + "<o:p></o:p></span></b></p>"
		// Contenido
		Content := response.PSEUDOCODE
		Content = strings.ReplaceAll(Content, "\n", "<br>")
		all = all + "<p class=MsoListParagraph style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.0pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph'><span lang=ES style='color:#241F0C;white-space: pre;'>" + Content + "<o:p></o:p></span></p>"
		all = all + "<br><br>"
		// Cadena de llamado
		all = all + "<h3><a name=\"_Toc40123021\"><span lang=ES>Cadena de llamados</span></a></h3>"
		all = all + "<p class=MsoNormal style='text-align:justify;text-justify:inter-ideograph;tab-stops:121.5pt'><span lang=ES style='font-family:\"Segoe UI Symbol\",sans-serif;mso-bidi-font-family:\"Segoe UI Symbol\";color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>&#10132; PRC01TEST<o:p></o:p></span></p>"
		all = all + "<br><br>"
		// Programas intervinientes
		all = all + "<h3><a name=\"_Toc40123022\"><span lang=ES>Programas intervinientes</span></a></h3>"
		all = all + "<p class=MsoListParagraphCxSpFirst style='margin-top:10.0pt;margin-right:0cm;margin-bottom:5.0pt;margin-left:36.25pt;mso-add-space:auto;text-align:justify;text-justify:inter-ideograph;text-indent:-18.0pt;mso-list:l33 level1 lfo49;tab-stops:121.5pt'><![if !supportLists]><span lang=ES style='font-family:Symbol;mso-fareast-font-family:Symbol;mso-bidi-font-family:Symbol;color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'><span style='mso-list:Ignore'>·<span style='font:7.0pt \"Times New Roman\"'>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span></span></span><![endif]><span lang=ES style='color:#241F0C;mso-fareast-language:ES;mso-bidi-font-weight:bold'>PRC01TEST<o:p></o:p></span></p>"

		rec := []byte(all)
		name := "Prueba ETP 2.htm"
		err = ioutil.WriteFile(name, rec, 0644)

		// Crear PDF
		wkhtmltopdf.SetPath(path)
		pdfg, err := wkhtmltopdf.NewPDFGenerator()

		if err != nil {
			log.Fatal(err)
		}
		page := wkhtmltopdf.NewPage("Prueba ETP 2.htm")

		pdfg.AddPage(page)

		// Create PDF document in internal buffer
		err = pdfg.Create()
		if err != nil {
			log.Fatal(err)
		}

		// Write buffer contents to file on disk
		err = pdfg.WriteFile("final.pdf")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Done")

		defer bottomFile.Close()
		defer topFile.Close()

		f, err := os.Open("final.pdf")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		fileInfo, err := f.Stat()
		if err != nil {
			panic(err)
		}

		http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), f)
	}
}

func buscarCod(textoSource string) string {
	fmt.Println("------------------- Buscando código -------------------")
	tInit := "<![CDATA["
	tFin := "]]>"
	rInit := 0
	rFin := 0
	tTemp := ""
	k := 0
	salir := ""
	for {
		// fmt.Println(i)
		if string(textoSource[k]) == "<" {
			tTemp = textoSource[k : k+len(tInit)]
			// fmt.Printf(strTemp)
			if tTemp == tInit {
				rInit = k
			}
		}
		if rInit != 0 {
			if string(textoSource[k]) == "]" {
				tTemp = textoSource[k : k+len(tFin)]
				// fmt.Println("strTemp: ", strTemp)
				if tTemp == tFin {
					rFin = k
					// fmt.Println("recortarI: ", rInit, "recortarF: ", rFin)
				}
			}
		}
		if rInit != 0 && rFin != 0 {
			// Buscar <Source> y </Source>
			salir = textoSource[rInit+len(tInit):rFin] + "\n"
			// fmt.Println(salir)
			break
		}
		k = k + 1
		if k == len(textoSource) {
			break
		}
	}
	return salir
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", homePage)
	myRouter.HandleFunc("/code", code).Methods("POST")
	myRouter.HandleFunc("/upload", upload).Methods("POST")
	log.Fatal(http.ListenAndServe(":4000", myRouter))
}

func main() {
	handleRequests()
}

func toUtf8(iso8859_1_buf []byte) string {
	buf := make([]rune, len(iso8859_1_buf))
	for i, b := range iso8859_1_buf {
		buf[i] = rune(b)
	}
	return string(buf)
}

func fromWindows1252(str string) string {
	var arr = []byte(str)
	var buf bytes.Buffer
	var r rune

	for _, b := range arr {
		switch b {
		case 0x80:
			r = 0x20AC
		case 0x82:
			r = 0x201A
		case 0x83:
			r = 0x0192
		case 0x84:
			r = 0x201E
		case 0x85:
			r = 0x2026
		case 0x86:
			r = 0x2020
		case 0x87:
			r = 0x2021
		case 0x88:
			r = 0x02C6
		case 0x89:
			r = 0x2030
		case 0x8A:
			r = 0x0160
		case 0x8B:
			r = 0x2039
		case 0x8C:
			r = 0x0152
		case 0x8E:
			r = 0x017D
		case 0x91:
			r = 0x2018
		case 0x92:
			r = 0x2019
		case 0x93:
			r = 0x201C
		case 0x94:
			r = 0x201D
		case 0x95:
			r = 0x2022
		case 0x96:
			r = 0x2013
		case 0x97:
			r = 0x2014
		case 0x98:
			r = 0x02DC
		case 0x99:
			r = 0x2122
		case 0x9A:
			r = 0x0161
		case 0x9B:
			r = 0x203A
		case 0x9C:
			r = 0x0153
		case 0x9E:
			r = 0x017E
		case 0x9F:
			r = 0x0178
		default:
			r = rune(b)
		}

		buf.WriteRune(r)
	}

	return string(buf.Bytes())
}
